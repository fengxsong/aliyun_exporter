package cmd

import (
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	"github.com/alecthomas/kingpin/v2"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	promlogflag "github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"

	"github.com/fengxsong/aliyun_exporter/pkg/collector"
	"github.com/fengxsong/aliyun_exporter/pkg/config"
	"github.com/fengxsong/aliyun_exporter/pkg/ratelimit"
)

type serveOption struct {
	configFile    string
	metricsPath   string
	rate          *int
	promlogConfig *promlog.Config
	webflagConfig *web.FlagConfig
}

func init() {
	commands["serve"] = &serveOption{}
}

func addPromlogConfig(f flagGroup) *promlog.Config {
	promlogConfig := &promlog.Config{
		Level:  &promlog.AllowedLevel{},
		Format: &promlog.AllowedFormat{},
	}
	f.Flag(promlogflag.LevelFlagName, promlogflag.LevelFlagHelp).Default("info").HintOptions(promlog.LevelFlagOptions...).SetValue(promlogConfig.Level)
	f.Flag(promlogflag.FormatFlagName, promlogflag.FormatFlagHelp).Default("logfmt").HintOptions(promlog.FormatFlagOptions...).SetValue(promlogConfig.Format)

	return promlogConfig
}

func (o *serveOption) help() string { return "Run http metrics handlers" }

func (o *serveOption) addFlags(f flagGroup) {
	f.Flag("config", "Filepath of config").Default("config.yaml").Short('c').StringVar(&o.configFile)
	f.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").StringVar(&o.metricsPath)

	o.promlogConfig = addPromlogConfig(f)
	o.webflagConfig = wrapWebFlagConfig(f, ":9527")
	o.rate = rateLimitFlag(f)
}

func wrapWebFlagConfig(f flagGroup, defaultAddress string) *web.FlagConfig {
	systemdSocket := func() *bool { b := false; return &b }() // Socket activation only available on Linux
	if runtime.GOOS == "linux" {
		systemdSocket = f.Flag(
			"web.systemd-socket",
			"Use systemd socket activation listeners instead of port listeners (Linux only).",
		).Bool()
	}
	flags := web.FlagConfig{
		WebListenAddresses: f.Flag(
			"web.listen-address",
			"Addresses on which to expose metrics and web interface. Repeatable for multiple addresses.",
		).Default(defaultAddress).Strings(),
		WebSystemdSocket: systemdSocket,
		WebConfigFile: f.Flag(
			"web.config.file",
			"Path to configuration file that can enable TLS or authentication. See: https://github.com/prometheus/exporter-toolkit/blob/master/docs/web-configuration.md",
		).Default("").String(),
	}
	return &flags
}

func (o *serveOption) run(_ *kingpin.ParseContext) error {
	cfg, err := config.Parse(o.configFile)
	if err != nil {
		return err
	}
	logger := promlog.New(o.promlogConfig)
	rt := ratelimit.New(*o.rate)
	cmc, err := collector.NewCloudMonitorCollector("cloudmonitor", cfg, rt, logger)
	if err != nil {
		return err
	}
	prometheus.MustRegister(cmc)

	if len(cfg.InstanceTypes) > 0 {
		level.Info(logger).Log("msg", "enabling instance info collectors for lable joining", "collectors", strings.Join(cfg.InstanceTypes, ", "))
		iic, err := collector.NewInstanceInfoCollector("cloudmonitor", cfg, rt, logger)
		if err != nil {
			return err
		}
		prometheus.MustRegister(iic)
	}

	http.Handle(o.metricsPath, promhttp.Handler())

	healthzPath := "/-/healthy"
	http.HandleFunc(healthzPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Healthy"))
	})

	landingConfig := web.LandingConfig{
		Name:        collector.Name(),
		Description: "a prometheus exporter for scraping metrics from alibaba cloudmonitor services",
		Version:     version.Info(),
		Links: []web.LandingLinks{
			{
				Address:     o.metricsPath,
				Text:        "Metrics",
				Description: "endpoint for scraping metrics",
			},
			{
				Address:     "/debug/pprof",
				Text:        "Pprof",
				Description: "pprof handlers",
			},
		},
	}

	landingpage, err := web.NewLandingPage(landingConfig)
	if err != nil {
		return err
	}
	http.Handle("/", landingpage)

	srv := &http.Server{}
	srvc := make(chan error)
	term := make(chan os.Signal, 1)
	signal.Notify(term, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := web.ListenAndServe(srv, o.webflagConfig, logger); err != nil {
			level.Error(logger).Log("msg", "Error starting HTTP server", "err", err)
			srvc <- err
			close(srvc)
		}
	}()

	for {
		select {
		case <-term:
			level.Info(logger).Log("msg", "Received SIGTERM, exiting gracefully...")
			return nil
		case err = <-srvc:
			return err
		}
	}
}

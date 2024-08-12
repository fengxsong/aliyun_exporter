package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/alecthomas/kingpin/v2"
	"github.com/go-kit/log/level"
	"github.com/prometheus/common/promlog"
	"gopkg.in/yaml.v2"

	"github.com/fengxsong/aliyun_exporter/pkg/client"
	"github.com/fengxsong/aliyun_exporter/pkg/ratelimit"
)

type flagGroup interface {
	Flag(string, string) *kingpin.FlagClause
}

type command interface {
	help() string
	run(*kingpin.ParseContext) error
}

var commands = map[string]command{}

func AddCommands(app *kingpin.Application) {
	for name, c := range commands {
		cc := app.Command(name, c.help())
		if flagRegisterer, ok := c.(interface {
			addFlags(flagGroup)
		}); ok {
			flagRegisterer.addFlags(cc)
		}
		cc.Action(c.run)
	}
}

func init() {
	commands["generate-example-config"] = &generateExampleConfigOption{}
}

func rateLimitFlag(f flagGroup) *int {
	return f.Flag("rate", "RPS/request per second to alibaba cloudmonitor service").Default("50").Int()
}

type generateExampleConfigOption struct {
	out           string
	ak, sk        string
	region        string
	includes      []string
	listing       bool
	rate          *int
	promlogConfig *promlog.Config
}

func (o *generateExampleConfigOption) help() string { return "Print example of config file" }

func (o *generateExampleConfigOption) addFlags(f flagGroup) {
	f.Flag("out", "Filepath to write example config to").Default("").Short('o').StringVar(&o.out)
	f.Flag("accesskey", "Access key").Envar("ALIBABA_CLOUD_ACCESS_KEY").Required().StringVar(&o.ak)
	f.Flag("accesskeysecret", "Access key secret").Envar("ALIBABA_CLOUD_ACCESS_KEY_SECRET").Required().StringVar(&o.sk)
	f.Flag("region", "Region id").Default("cn-hangzhou").StringVar(&o.region)
	f.Flag("includes", "Only print metrics list of specified namespaces, default will print all").Default("").StringsVar(&o.includes)
	f.Flag("list", "List avaliable namespaces of metrics only").Default("false").Short('l').BoolVar(&o.listing)

	o.promlogConfig = addPromlogConfig(f)
	o.rate = rateLimitFlag(f)
}

func (o *generateExampleConfigOption) run(_ *kingpin.ParseContext) error {
	rt := ratelimit.New(*o.rate)
	logger := promlog.New(o.promlogConfig)
	mc, err := client.NewMetricClient(o.ak, o.sk, o.region, rt, logger)
	if err != nil {
		return err
	}
	if o.listing {
		namespaces, err := mc.ListingNamespace()
		if err != nil {
			return err
		}
		level.Info(logger).Log("msg", fmt.Sprintf("available namespaces are %s", namespaces))
		return nil
	}
	if len(o.includes) == 1 && o.includes[0] == "" {
		o.includes = []string{}
	}
	cfg, err := mc.GenerateExampleConfig(o.ak, o.sk, o.region, o.includes...)
	if err != nil {
		return err
	}
	level.Info(logger).Log("msg", fmt.Sprintf("the builtin instance info collectors are %s, feel free to summit a PR", strings.Join(cfg.InstanceTypes, ", ")))
	var writer io.Writer
	switch o.out {
	case "", "stdout":
		writer = os.Stdout
	default:
		writer, err = os.Create(o.out)
		if err != nil {
			return err
		}
	}
	if err = yaml.NewEncoder(writer).Encode(&cfg); err != nil {
		return err
	}
	level.Info(logger).Log("msg", "example configurations have been successfully generated, please modify the corresponding 'period'/'measure'/'instance_types' fields before running.")
	return nil
}

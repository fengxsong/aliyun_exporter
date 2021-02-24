package cmd

import (
	"os"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/spf13/cobra"
)

var logger log.Logger

// options command line options
type options struct {
	logFmt    string
	logLevel  string
	logFile   string
	rateLimit int
	so        *serveOption
	pco       *printConfigOption
}

func (o *options) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&o.logFmt, "log.format", "logfmt", "Output format of log messages. One of: [logfmt, json]")
	cmd.Flags().StringVar(&o.logLevel, "log.level", "info", "Log level")
	cmd.Flags().StringVar(&o.logFile, "log.file", "", "Log message to file")
	cmd.Flags().IntVar(&o.rateLimit, "rate-limit", 1<<8, "RPS/request per second")
	if o.so != nil {
		o.so.AddFlags(cmd)
	}
	if o.pco != nil {
		o.pco.AddFlags(cmd)
	}
}

// Complete do some initialization
func (o *options) Complete() error {
	writer := os.Stdout
	if o.logFile != "" {
		out, err := os.OpenFile(o.logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		writer = out
	}
	switch o.logFmt {
	case "logfmt":
		logger = log.NewLogfmtLogger(writer)
	case "json":
		logger = log.NewJSONLogger(writer)
	default:
		logger = log.NewNopLogger()
	}
	var lvlOp level.Option
	switch strings.ToLower(o.logLevel) {
	case "debug":
		lvlOp = level.AllowDebug()
	case "info":
		lvlOp = level.AllowInfo()
	case "warn", "warning":
		lvlOp = level.AllowWarn()
	case "error":
		lvlOp = level.AllowError()
	default:
		level.Info(logger).Log("msg", "unknown log level, fallback to info", "level", o.logLevel)
		lvlOp = level.AllowInfo()
	}
	logger = level.NewFilter(logger, lvlOp)
	if o.so != nil {
		if err := o.so.Complete(); err != nil {
			return err
		}
	}
	if o.pco != nil {
		if err := o.pco.Complete(); err != nil {
			return err
		}
	}
	return nil
}

type serveOption struct {
	configFile    string
	listenAddress string
	metricPath    string
}

func (o *serveOption) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&o.configFile, "config", "c", "config.yaml", "Path of config file")
	cmd.Flags().StringVar(&o.listenAddress, "web.listen-address", ":9527", "Address on which to expose metrics and web interface.")
	cmd.Flags().StringVar(&o.metricPath, "web.telemetry-path", "/metrics", "Path under which to expose metrics")
}

func (o *serveOption) Complete() error {
	return nil
}

type printConfigOption struct {
	ak     string
	secret string
	region string
	out    string
}

func (o *printConfigOption) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&o.ak, "ak", "", "Access key")
	cmd.MarkFlagRequired("ak")
	cmd.Flags().StringVar(&o.secret, "secret", "", "Access key secret")
	cmd.MarkFlagRequired("secret")
	cmd.Flags().StringVar(&o.region, "region", "cn-hangzhou", "Region ID")
	cmd.Flags().StringVarP(&o.out, "out", "o", "", "Path of file to write config")
}

func (o *printConfigOption) Complete() error {
	return nil
}

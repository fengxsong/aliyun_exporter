package cmd

import (
	"os"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

// Options command line options
type Options struct {
	configFile    string
	listenAddress string
	metricPath    string
	logFmt        string
	logLevel      string
	logFile       string
	rateLimit     int
}

var logger log.Logger

// Complete do some initialization
func (o *Options) Complete() error {
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
	return nil
}

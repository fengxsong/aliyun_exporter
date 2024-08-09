package main

import (
	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
	versioncollector "github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/common/version"

	"github.com/fengxsong/aliyun_exporter/cmd"
	"github.com/fengxsong/aliyun_exporter/pkg/collector"
)

func init() {
	prometheus.MustRegister(versioncollector.NewCollector(collector.Name()))
}

func main() {
	kingpin.Version(version.Print(collector.Name()))
	kingpin.HelpFlag.Short('h')
	cmd.AddCommands(kingpin.CommandLine)
	kingpin.Parse()
}

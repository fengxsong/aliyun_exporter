package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	"github.com/fengxsong/aliyun-exporter/pkg/client"
	"github.com/fengxsong/aliyun-exporter/pkg/collector"
	"github.com/fengxsong/aliyun-exporter/pkg/config"
	"github.com/fengxsong/aliyun-exporter/pkg/handler"
	"github.com/fengxsong/aliyun-exporter/pkg/ratelimit"
	"github.com/fengxsong/aliyun-exporter/version"
)

const appName = "cloudmonitor"

// NewRootCommand create root command
func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           appName,
		Short:         "Exporter for aliyun cloudmonitor",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	cmd.AddCommand(newServeMetricsCommand())
	cmd.AddCommand(newVersionCommand())
	cmd.AddCommand(printConfigCommand())
	cmd.AddCommand(newListMetricNamespacesCommand())
	return cmd
}

func newServeMetricsCommand() *cobra.Command {
	o := &Options{}
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve HTTP metrics handler",
		PreRunE: func(_ *cobra.Command, _ []string) error {
			return o.Complete()
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := config.Parse(o.configFile)
			if err != nil {
				return err
			}
			rt := ratelimit.New(o.rateLimit)
			cm, err := collector.NewCloudMonitorCollector(appName, cfg, rt, logger)
			if err != nil {
				return err
			}
			iif, err := collector.NewInstanceInfoCollector(appName, cfg, rt, logger)
			if err != nil {
				return err
			}
			h, err := handler.New(o.listenAddress, o.metricPath, logger, cm, iif)
			if err != nil {
				return err
			}
			return h.Run()
		},
	}
	cmd.Flags().StringVarP(&o.configFile, "config", "c", "config.yaml", "Path of config file")
	cmd.Flags().StringVar(&o.listenAddress, "web.listen-address", ":9527", "Address on which to expose metrics and web interface.")
	cmd.Flags().StringVar(&o.metricPath, "web.telemetry-path", "/metrics", "Path under which to expose metrics")
	cmd.Flags().StringVar(&o.logFmt, "log.format", "logfmt", "Output format of log messages. One of: [logfmt, json]")
	cmd.Flags().StringVar(&o.logLevel, "log.level", "info", "Log level")
	cmd.Flags().StringVar(&o.logFile, "log.file", "", "Log message to file")
	cmd.Flags().IntVar(&o.rateLimit, "rate-limit", 1<<8, "RPS/request per second")
	return cmd
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version info",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println(version.Version())
		},
	}
}

func newListMetricNamespacesCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list-metrics",
		Short: "List avaliable namespaces of metrics",
		Run: func(_ *cobra.Command, _ []string) {
			w := tabwriter.NewWriter(os.Stdout, 0, 8, 0, '\t', 0)
			fmt.Fprintln(w, "NAMESPACE\tDESCRIPTION")
			for name, desc := range client.AllNamespaces() {
				fmt.Fprintf(w, "%s\t%s\n", name, desc)
			}
			w.Flush()
		},
	}
}

func printConfigCommand() *cobra.Command {
	var (
		ak, secret, region string
		out                string
		rateLimit          int
	)
	cmd := &cobra.Command{
		Use:   "print-config",
		Short: "Print example of config file",
		RunE: func(_ *cobra.Command, args []string) error {
			rt := ratelimit.New(rateLimit)
			cm, err := client.NewMetricClient(ak, secret, region, rt, logger)
			if err != nil {
				return err
			}
			metaMap, err := cm.DescribeMetricMetaList(args...)
			if err != nil {
				return err
			}
			cfg := client.GenerateExampleConfig(ak, secret, region, metaMap)
			b, err := yaml.Marshal(&cfg)
			if err != nil {
				return err
			}
			if len(out) == 0 {
				fmt.Println(string(b))
				return nil
			}
			w, err := ioutil.TempFile("", "config")
			if err != nil {
				return nil
			}
			defer w.Close()
			w.Write(b)
			return os.Rename(w.Name(), out)
		},
	}
	cmd.Flags().StringVar(&ak, "ak", "", "Access key")
	cmd.MarkFlagRequired("ak")
	cmd.Flags().StringVar(&secret, "secret", "", "Access key secret")
	cmd.MarkFlagRequired("secret")
	cmd.Flags().StringVar(&region, "region", "cn-hangzhou", "Region ID")
	cmd.Flags().StringVarP(&out, "out", "o", "", "Path of file to write config")
	cmd.Flags().IntVar(&rateLimit, "rate-limit", 1<<8, "RPS/request per second")
	return cmd
}

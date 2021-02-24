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
	o := &options{
		so: &serveOption{},
	}
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve HTTP metrics handler",
		PreRunE: func(_ *cobra.Command, _ []string) error {
			return o.Complete()
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := config.Parse(o.so.configFile)
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
			h, err := handler.New(o.so.listenAddress, o.so.metricPath, logger, cm, iif)
			if err != nil {
				return err
			}
			return h.Run()
		},
	}
	o.AddFlags(cmd)
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
	o := &options{
		pco: &printConfigOption{},
	}
	cmd := &cobra.Command{
		Use:   "print-config",
		Short: "Print example of config file",
		PreRunE: func(_ *cobra.Command, _ []string) error {
			return o.Complete()
		},
		RunE: func(_ *cobra.Command, args []string) error {
			rt := ratelimit.New(o.rateLimit)
			cm, err := client.NewMetricClient(o.pco.ak, o.pco.secret, o.pco.region, rt, logger)
			if err != nil {
				return err
			}
			metaMap, err := cm.DescribeMetricMetaList(args...)
			if err != nil {
				return err
			}
			cfg := client.GenerateExampleConfig(o.pco.ak, o.pco.secret, o.pco.region, metaMap)
			b, err := yaml.Marshal(&cfg)
			if err != nil {
				return err
			}
			if len(o.pco.out) == 0 {
				fmt.Println(string(b))
				return nil
			}
			w, err := ioutil.TempFile("", "config")
			if err != nil {
				return nil
			}
			defer w.Close()
			w.Write(b)
			return os.Rename(w.Name(), o.pco.out)
		},
	}
	o.AddFlags(cmd)
	return cmd
}

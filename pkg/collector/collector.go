package collector

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/fengxsong/aliyun-exporter/pkg/client"
	"github.com/fengxsong/aliyun-exporter/pkg/config"
)

// cloudMonitor ..
type cloudMonitor struct {
	namespace string
	cfg       *config.Config
	logger    log.Logger
	// sdk client
	client *client.MetricClient
	// collector metrics
	scrapeDurationDesc *prometheus.Desc

	lock sync.Mutex
}

// NewCloudMonitorCollector create a new collector for cloud monitor
func NewCloudMonitorCollector(appName string, cfg *config.Config, rt http.RoundTripper, logger log.Logger) (prometheus.Collector, error) {
	if logger == nil {
		logger = log.NewNopLogger()
	}
	cli, err := client.NewMetricClient(cfg.AccessKey, cfg.AccessKeySecret, cfg.Region, rt, logger)
	if err != nil {
		return nil, err
	}
	return &cloudMonitor{
		namespace: appName,
		cfg:       cfg,
		logger:    logger,
		client:    cli,
		scrapeDurationDesc: prometheus.NewDesc(
			prometheus.BuildFQName(appName, "scrape", "collector_duration_seconds"),
			fmt.Sprintf("%s_exporter: Duration of a collector scrape.", appName),
			[]string{"namespace", "collector"},
			nil,
		),
	}, nil
}

func (m *cloudMonitor) Describe(ch chan<- *prometheus.Desc) {
	ch <- m.scrapeDurationDesc
}

func (m *cloudMonitor) Collect(ch chan<- prometheus.Metric) {
	m.lock.Lock()
	defer m.lock.Unlock()

	wg := &sync.WaitGroup{}
	// do collect
	for sub, metrics := range m.cfg.Metrics {
		for i := range metrics {
			wg.Add(1)
			go func(namespace string, metric *config.Metric) {
				defer wg.Done()
				start := time.Now()
				m.client.Collect(m.namespace, namespace, metric, ch)
				ch <- prometheus.MustNewConstMetric(
					m.scrapeDurationDesc,
					prometheus.GaugeValue,
					time.Now().Sub(start).Seconds(),
					namespace, metric.String())
			}(sub, metrics[i])
		}
	}
	wg.Wait()
}

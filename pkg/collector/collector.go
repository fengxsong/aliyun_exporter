package collector

import (
	"net/http"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/fengxsong/aliyun_exporter/pkg/client"
	"github.com/fengxsong/aliyun_exporter/pkg/config"
)

const app = "aliyun_exporter"

func Name() string { return app }

var (
	scrapeDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: app,
		Name:      "scrape_duration_seconds",
		Help:      "Duration of each metrics scraping"},
		[]string{"namespace", "collector", "region"})
	scrapeTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: app,
		Name:      "scrape_total",
		Help:      "Total scrape counts",
	}, []string{"namespace", "collector", "region", "state"})
)

func init() {
	prometheus.MustRegister(scrapeDuration, scrapeTotal)
}

// cloudMonitor ..
type cloudMonitor struct {
	// namespace is the prefix of all registered metrics
	namespace string
	cfg       *config.Config
	logger    log.Logger
	client    *client.MetricClient
	lock      sync.Mutex
}

// NewCloudMonitorCollector create a new collector for cloud monitor
func NewCloudMonitorCollector(namespace string, cfg *config.Config, rt http.RoundTripper, logger log.Logger) (prometheus.Collector, error) {
	if logger == nil {
		logger = log.NewNopLogger()
	}
	cli, err := client.NewMetricClient(cfg.AccessKey, cfg.AccessKeySecret, cfg.Region, rt, logger)
	if err != nil {
		return nil, err
	}
	c := &cloudMonitor{
		namespace: namespace,
		cfg:       cfg,
		logger:    logger,
		client:    cli,
	}
	return c, nil
}

func (m *cloudMonitor) Describe(ch chan<- *prometheus.Desc) {}

func (m *cloudMonitor) Collect(ch chan<- prometheus.Metric) {
	m.lock.Lock()
	defer m.lock.Unlock()

	wg := &sync.WaitGroup{}
	for sub, metrics := range m.cfg.Metrics {
		for i := range metrics {
			wg.Add(1)
			go func(namespace string, metric *config.Metric) {
				start := time.Now()
				defer func() {
					scrapeDuration.WithLabelValues(namespace, metric.String(), m.cfg.Region).Observe(time.Since(start).Seconds())
					wg.Done()
				}()
				if err := m.client.Collect(m.namespace, namespace, metric, ch); err != nil {
					level.Error(m.logger).Log("err", err, "sub", namespace, "metric", metric.String())
					scrapeTotal.WithLabelValues(namespace, metric.String(), m.cfg.Region, "failed").Inc()
					return
				}
				scrapeTotal.WithLabelValues(namespace, metric.String(), m.cfg.Region, "success").Inc()
			}(sub, metrics[i])
		}
	}
	wg.Wait()
}

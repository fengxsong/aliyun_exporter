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

type instanceInfoCollector struct {
	namespace string
	cfg       *config.Config
	logger    log.Logger
	client    *client.ServiceClient
	lock      sync.Mutex
}

// NewInstanceInfoCollector create a new collector for instance info
func NewInstanceInfoCollector(namespace string, cfg *config.Config, rt http.RoundTripper, logger log.Logger) (prometheus.Collector, error) {
	if logger == nil {
		logger = log.NewNopLogger()
	}
	cli, err := client.NewServiceClient(cfg.AccessKey, cfg.AccessKeySecret, cfg.Region, rt, logger)
	if err != nil {
		return nil, err
	}
	return &instanceInfoCollector{
		namespace: namespace,
		cfg:       cfg,
		logger:    logger,
		client:    cli,
	}, nil
}

func (c *instanceInfoCollector) Describe(ch chan<- *prometheus.Desc) {}

func (c *instanceInfoCollector) Collect(ch chan<- prometheus.Metric) {
	c.lock.Lock()
	defer c.lock.Unlock()
	wg := &sync.WaitGroup{}
	for i := range c.cfg.InstanceTypes {
		wg.Add(1)
		go func(sub string) {
			start := time.Now()
			defer func() {
				wg.Done()
				scrapeDuration.WithLabelValues(sub, "InstanceInfo").Observe(time.Since(start).Seconds())
			}()
			if err := c.client.Collect(c.namespace, sub, ch); err != nil {
				level.Error(c.logger).Log("err", err, "instancetype", sub)
				scrapeTotal.WithLabelValues(sub, "InstanceInfo", "failed").Inc()
				return
			}
			scrapeTotal.WithLabelValues(sub, "InstanceInfo", "success").Inc()
		}(c.cfg.InstanceTypes[i])
	}
	wg.Wait()
}

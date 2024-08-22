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
	cli, err := client.NewServiceClient(cfg.AccessKey, cfg.AccessKeySecret, cfg.InstanceInfo.Regions, rt, logger)
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
	for i := range c.cfg.InstanceInfo.Types {
		for j := range c.cfg.InstanceInfo.Regions {
			wg.Add(1)
			go func(sub string, region string) {
				start := time.Now()
				defer func() {
					wg.Done()
					scrapeDuration.WithLabelValues(sub, "InstanceInfo", region).Observe(time.Since(start).Seconds())
				}()
				if err := c.client.Collect(c.namespace, sub, region, ch); err != nil {
					level.Error(c.logger).Log("err", err, "instancetype", sub)
					scrapeTotal.WithLabelValues(sub, "InstanceInfo", region, "failed").Inc()
					return
				}
				scrapeTotal.WithLabelValues(sub, "InstanceInfo", region, "success").Inc()
			}(c.cfg.InstanceInfo.Types[i], c.cfg.InstanceInfo.Regions[j])
		}
	}
	wg.Wait()
}

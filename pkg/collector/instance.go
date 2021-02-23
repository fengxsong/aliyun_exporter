package collector

import (
	"net/http"
	"sync"

	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/fengxsong/aliyun-exporter/pkg/client"
	"github.com/fengxsong/aliyun-exporter/pkg/config"
)

type instanceInfo struct {
	namespace string
	cfg       *config.Config
	logger    log.Logger
	// sdk client
	client *client.ServiceClient
	lock   sync.Mutex
}

// NewInstanceInfoCollector create a new collector for instance info
func NewInstanceInfoCollector(appName string, cfg *config.Config, rt http.RoundTripper, logger log.Logger) (prometheus.Collector, error) {
	if logger == nil {
		logger = log.NewNopLogger()
	}
	cli, err := client.NewServiceClient(cfg.AccessKey, cfg.AccessKeySecret, cfg.Region, rt, logger)
	if err != nil {
		return nil, err
	}
	return &instanceInfo{
		namespace: appName,
		cfg:       cfg,
		logger:    logger,
		client:    cli,
	}, nil
}

func (iif *instanceInfo) Describe(ch chan<- *prometheus.Desc) {
}

func (iif *instanceInfo) Collect(ch chan<- prometheus.Metric) {
	iif.lock.Lock()
	defer iif.lock.Unlock()
	wg := &sync.WaitGroup{}
	for i := range iif.cfg.InstanceInfos {
		wg.Add(1)
		go func(s string) {
			defer wg.Done()
			iif.client.Collect(iif.namespace, s, ch)
		}(iif.cfg.InstanceInfos[i])
	}
	wg.Wait()
}

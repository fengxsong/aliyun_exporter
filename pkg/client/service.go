package client

import (
	"net/http"

	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"

	// register functions
	_ "github.com/fengxsong/aliyun-exporter/pkg/client/dds"
	_ "github.com/fengxsong/aliyun-exporter/pkg/client/ecs"
	_ "github.com/fengxsong/aliyun-exporter/pkg/client/es"
	_ "github.com/fengxsong/aliyun-exporter/pkg/client/rds"
	_ "github.com/fengxsong/aliyun-exporter/pkg/client/redis"
	_ "github.com/fengxsong/aliyun-exporter/pkg/client/slb"

	"github.com/fengxsong/aliyun-exporter/pkg/client/service"
)

// ServiceClient ...
type ServiceClient struct {
	collectors map[string]service.Collector
}

// Collect do actural collection
func (c *ServiceClient) Collect(namespace string, sub string, ch chan<- prometheus.Metric) {
	collector, ok := c.collectors[sub]
	if !ok {
		return
	}
	collector.Collect(namespace, ch)
}

// NewServiceClient create service client
func NewServiceClient(ak, secret, region string, rt http.RoundTripper, logger log.Logger) (*ServiceClient, error) {
	sc := &ServiceClient{
		collectors: make(map[string]service.Collector),
	}
	if logger == nil {
		logger = log.NewNopLogger()
	}
	for name, fn := range service.CollectorFunc() {
		collector, err := fn(ak, secret, region, rt, logger)
		if err != nil {
			return nil, err
		}
		sc.collectors[name] = collector
	}
	return sc, nil
}

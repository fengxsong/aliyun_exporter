package client

import (
	"net/http"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/fengxsong/aliyun_exporter/pkg/client/services"
)

// ServiceClient ...
type ServiceClient struct {
	collectors map[string]services.Collector
}

// Collect do actural collection
func (c *ServiceClient) Collect(namespace string, sub string, region string, ch chan<- prometheus.Metric) error {
	key := sub + region
	collector, ok := c.collectors[key]
	if !ok {
		return nil
	}
	return collector.Collect(namespace, ch)
}

// NewServiceClient create service client
func NewServiceClient(ak, secret string, regions []string, rt http.RoundTripper, logger log.Logger) (*ServiceClient, error) {
	sc := &ServiceClient{
		collectors: make(map[string]services.Collector),
	}
	if logger == nil {
		logger = log.NewNopLogger()
	}
	for name, fn := range services.All() {
		for i := range regions {
			collector, err := fn(regions[i], ak, secret, logger)
			if err != nil {
				return nil, err
			}
			if client, ok := collector.(interface {
				SetTransport(http.RoundTripper)
			}); ok {
				client.SetTransport(rt)
			}
			sc.collectors[name+regions[i]] = collector
		}
	}
	return sc, nil
}

package slb

import (
	"net/http"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/slb"
	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/fengxsong/aliyun-exporter/pkg/client/service"
)

// constants
const (
	name     = "slb"
	pageSize = 100
)

// Client wrap client
type Client struct {
	*slb.Client
	desc   *prometheus.Desc
	logger log.Logger
}

// New create ServiceCollector
func New(ak, secret, region string, rt http.RoundTripper, logger log.Logger) (service.Collector, error) {
	client, err := slb.NewClientWithAccessKey(region, ak, secret)
	if err != nil {
		return nil, err
	}
	client.SetTransport(rt)
	return &Client{Client: client, logger: logger}, nil
}

// Collect collect metrics
func (c *Client) Collect(namespace string, ch chan<- prometheus.Metric) {
	if c.desc == nil {
		c.desc = service.NewInstanceClientDesc(namespace, name, []string{"regionId", "instanceId", "name", "address", "status"})
	}
	req := slb.CreateDescribeLoadBalancersRequest()
	req.PageSize = requests.NewInteger(pageSize)
	instanceCh := make(chan slb.LoadBalancer, 1<<10)
	go func() {
		defer close(instanceCh)
		for hasNextPage, pageNum := true, 1; hasNextPage != false; pageNum++ {
			req.PageNumber = requests.NewInteger(pageNum)
			response, err := c.DescribeLoadBalancers(req)
			if err != nil {
				return
			}
			if len(response.LoadBalancers.LoadBalancer) < pageSize {
				hasNextPage = false
			}
			for i := range response.LoadBalancers.LoadBalancer {
				instanceCh <- response.LoadBalancers.LoadBalancer[i]
			}
		}
	}()

	for ins := range instanceCh {
		ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, 1.0,
			ins.RegionId, ins.LoadBalancerId, ins.LoadBalancerName, ins.Address, ins.LoadBalancerStatus)
	}
}

func init() {
	service.Register(name, New)
}

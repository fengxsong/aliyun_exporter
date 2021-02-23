package redis

import (
	"net/http"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	r_kvstore "github.com/aliyun/alibaba-cloud-sdk-go/services/r-kvstore"
	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/fengxsong/aliyun-exporter/pkg/client/service"
)

// constants
const (
	name     = "redis"
	pageSize = 100
)

// Client wrap client
type Client struct {
	*r_kvstore.Client
	desc   *prometheus.Desc
	logger log.Logger
}

// New create ServiceCollector
func New(ak, secret, region string, rt http.RoundTripper, logger log.Logger) (service.Collector, error) {
	client, err := r_kvstore.NewClientWithAccessKey(region, ak, secret)
	if err != nil {
		return nil, err
	}
	client.SetTransport(rt)
	return &Client{Client: client, logger: logger}, nil
}

// Collect collect metrics
func (c *Client) Collect(namespace string, ch chan<- prometheus.Metric) {
	if c.desc == nil {
		c.desc = service.NewInstanceClientDesc(namespace, name, []string{"regionId", "instanceId", "name", "connectionDomain", "address", "status"})
	}
	req := r_kvstore.CreateDescribeInstancesRequest()
	req.PageSize = requests.NewInteger(pageSize)
	instanceCh := make(chan r_kvstore.KVStoreInstance, 1<<10)
	go func() {
		defer close(instanceCh)
		for hasNextPage, pageNum := true, 1; hasNextPage != false; pageNum++ {
			req.PageNumber = requests.NewInteger(pageNum)
			response, err := c.DescribeInstances(req)
			if err != nil {
				return
			}
			if len(response.Instances.KVStoreInstance) < pageSize {
				hasNextPage = false
			}
			for i := range response.Instances.KVStoreInstance {
				instanceCh <- response.Instances.KVStoreInstance[i]
			}
		}
	}()

	for ins := range instanceCh {
		ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, 1.0,
			ins.RegionId, ins.InstanceId, ins.InstanceName, ins.ConnectionDomain, ins.PrivateIp, ins.InstanceStatus)
	}
}

func init() {
	service.Register(name, New)
}

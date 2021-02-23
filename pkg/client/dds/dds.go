package dds

import (
	"net/http"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/dds"
	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/fengxsong/aliyun-exporter/pkg/client/service"
)

// constants
const (
	name     = "mongo"
	pageSize = 100
)

// Client wrap client
type Client struct {
	*dds.Client
	desc   *prometheus.Desc
	logger log.Logger
}

// New create ServiceCollector
func New(ak, secret, region string, rt http.RoundTripper, logger log.Logger) (service.Collector, error) {
	client, err := dds.NewClientWithAccessKey(region, ak, secret)
	if err != nil {
		return nil, err
	}
	client.SetTransport(rt)
	return &Client{Client: client, logger: logger}, nil
}

// Collect collect metrics
func (c *Client) Collect(namespace string, ch chan<- prometheus.Metric) {
	if c.desc == nil {
		c.desc = service.NewInstanceClientDesc(namespace, name, []string{"regionId", "instanceId", "dbType", "desc", "status"})
	}

	req := dds.CreateDescribeDBInstancesRequest()
	req.PageSize = requests.NewInteger(pageSize)
	instanceCh := make(chan dds.DBInstance, 1<<10)
	go func() {
		defer close(instanceCh)
		for hasNextPage, pageNum := true, 1; hasNextPage != false; pageNum++ {
			req.PageNumber = requests.NewInteger(pageNum)
			response, err := c.DescribeDBInstances(req)
			if err != nil {
				return
			}
			if len(response.DBInstances.DBInstance) < pageSize {
				hasNextPage = false
			}
			for i := range response.DBInstances.DBInstance {
				instanceCh <- response.DBInstances.DBInstance[i]
			}
		}
	}()
	for ins := range instanceCh {
		ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, 1.0,
			ins.RegionId, ins.DBInstanceId, ins.DBInstanceType, ins.DBInstanceDescription, ins.DBInstanceStatus)
	}
}

func init() {
	service.Register(name, New)
}

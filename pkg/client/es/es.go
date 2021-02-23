package es

import (
	"net/http"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/elasticsearch"
	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/fengxsong/aliyun-exporter/pkg/client/service"
)

// constants
const (
	name     = "es"
	pageSize = 100
)

// Client wrap client
type Client struct {
	*elasticsearch.Client
	desc   *prometheus.Desc
	logger log.Logger
}

// New create ServiceCollector
func New(ak, secret, region string, rt http.RoundTripper, logger log.Logger) (service.Collector, error) {
	client, err := elasticsearch.NewClientWithAccessKey(region, ak, secret)
	if err != nil {
		return nil, err
	}
	client.SetTransport(rt)
	return &Client{Client: client, logger: logger}, nil
}

// Collect collect metrics
func (c *Client) Collect(namespace string, ch chan<- prometheus.Metric) {
	if c.desc == nil {
		c.desc = service.NewInstanceClientDesc(namespace, name, []string{"instanceId", "desc", "status"})
	}

	req := elasticsearch.CreateListInstanceRequest()
	req.Size = requests.NewInteger(pageSize)
	instanceCh := make(chan elasticsearch.Instance, 1<<10)
	go func() {
		defer close(instanceCh)
		for hasNextPage, pageNum := true, 1; hasNextPage != false; pageNum++ {
			req.Page = requests.NewInteger(pageNum)
			response, err := c.ListInstance(req)
			if err != nil {
				return
			}
			if len(response.Result) < pageSize {
				hasNextPage = false
			}
			for i := range response.Result {
				instanceCh <- response.Result[i]
			}
		}
	}()

	for ins := range instanceCh {
		ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, 1.0,
			ins.InstanceId, ins.Description, ins.Status)
	}
}

func init() {
	service.Register(name, New)
}

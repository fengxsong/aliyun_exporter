package services

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/slb"
	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
)

// Client wrap client
type slbClient struct {
	*slb.Client
	desc   *prometheus.Desc
	logger log.Logger
}

// Collect collect metrics
func (c *slbClient) Collect(namespace string, ch chan<- prometheus.Metric) error {
	if c.desc == nil {
		c.desc = newDescOfInstnaceInfo(namespace, "acs_slb_dashboard",
			[]string{"regionId", "instanceId", "name", "address", "type", "status"})
	}
	req := slb.CreateDescribeLoadBalancersRequest()
	req.PageSize = requests.NewInteger(pageSize)
	resultChan := make(chan *result, 1<<10)
	var totalCount int

	go func() {
		defer close(resultChan)
		for hasNextPage, pageNum := true, 1; hasNextPage != false; pageNum++ {
			req.PageNumber = requests.NewInteger(pageNum)
			response, err := c.DescribeLoadBalancers(req)
			if err != nil {
				resultChan <- &result{err: err}
				return
			}
			totalCount += len(response.LoadBalancers.LoadBalancer)
			if totalCount >= response.TotalCount {
				hasNextPage = false
			}
			for i := range response.LoadBalancers.LoadBalancer {
				resultChan <- &result{v: response.LoadBalancers.LoadBalancer[i]}
			}
		}
	}()

	for res := range resultChan {
		if res.err != nil {
			return res.err
		}
		ins := res.v.(slb.LoadBalancer)
		ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, 1.0,
			ins.RegionId, ins.LoadBalancerId, ins.LoadBalancerName, ins.Address, ins.SpecType, ins.LoadBalancerStatus)
	}
	return nil
}

func init() {
	register("acs_slb_dashboard", func(s1, s2, s3 string, l log.Logger) (Collector, error) {
		client, err := slb.NewClientWithAccessKey(s1, s2, s3)
		if err != nil {
			return nil, err
		}
		return &slbClient{Client: client, logger: l}, nil
	})
}

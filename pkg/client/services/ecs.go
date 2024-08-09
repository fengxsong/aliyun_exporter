package services

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
)

type ecsClient struct {
	*ecs.Client
	desc   *prometheus.Desc
	logger log.Logger
}

func (c *ecsClient) Collect(namespace string, ch chan<- prometheus.Metric) error {
	if c.desc == nil {
		c.desc = newDescOfInstnaceInfo(namespace, "ecs", []string{"regionId", "instanceId", "instanceName", "hostname", "status"})
	}

	req := ecs.CreateDescribeInstancesRequest()
	req.PageSize = requests.NewInteger(pageSize)
	resultChan := make(chan *result, 1<<10)

	go func() {
		defer close(resultChan)
		for hasNextPage, pageNum := true, 1; hasNextPage != false; pageNum++ {
			req.PageNumber = requests.NewInteger(pageNum)
			req.PageSize = requests.NewInteger(pageSize)
			response, err := c.DescribeInstances(req)
			if err != nil {
				resultChan <- &result{err: err}
				return
			}
			if len(response.Instances.Instance) < pageSize {
				hasNextPage = false
			}
			for i := range response.Instances.Instance {
				resultChan <- &result{v: response.Instances.Instance[i]}
			}
		}
	}()
	for res := range resultChan {
		if res.err != nil {
			return res.err
		}
		ins := res.v.(ecs.Instance)
		ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, 1.0,
			ins.RegionId, ins.InstanceId, ins.InstanceName, ins.HostName, ins.Status)
	}
	return nil
}

func init() {
	register("ecs", func(s1, s2, s3 string, l log.Logger) (Collector, error) {
		client, err := ecs.NewClientWithAccessKey(s1, s2, s3)
		if err != nil {
			return nil, err
		}
		return &ecsClient{Client: client, logger: l}, nil
	})
}

package services

import (
	kafka "github.com/aliyun/alibaba-cloud-sdk-go/services/alikafka"
	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
)

type kafkaClient struct {
	*kafka.Client
	desc   *prometheus.Desc
	logger log.Logger
}

func (c *kafkaClient) Collect(namespace string, ch chan<- prometheus.Metric) error {
	if c.desc == nil {
		c.desc = newDescOfInstnaceInfo(namespace, "acs_kafka", []string{"regionId", "instanceId", "name", "endpoint", "type"})
	}

	req := kafka.CreateGetInstanceListRequest()
	resultChan := make(chan *result, 1<<10)

	go func() {
		defer close(resultChan)
		for hasNextPage, pageNum := true, 1; hasNextPage != false; pageNum++ {
			response, err := c.GetInstanceList(req)
			if err != nil {
				resultChan <- &result{err: err}
				return
			}
			if len(response.InstanceList.InstanceVO) < pageSize && len(response.InstanceList.InstanceVO) == 0 {
				hasNextPage = false
			}
			for i := range response.InstanceList.InstanceVO {
				resultChan <- &result{v: response.InstanceList.InstanceVO[i]}
			}
		}
	}()
	for res := range resultChan {
		if res.err != nil {
			return res.err
		}
		ins := res.v.(kafka.InstanceVO)
		ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, 1.0,
			ins.RegionId, ins.InstanceId, ins.Name, ins.DomainEndpoint, ins.SpecType)
	}
	return nil
}

func init() {
	register("acs_kafka", func(s1, s2, s3 string, l log.Logger) (Collector, error) {
		client, err := kafka.NewClientWithAccessKey(s1, s2, s3)
		if err != nil {
			return nil, err
		}
		return &kafkaClient{Client: client, logger: l}, nil
	})
}

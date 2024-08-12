package services

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	r_kvstore "github.com/aliyun/alibaba-cloud-sdk-go/services/r-kvstore"
	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
)

type redisClient struct {
	*r_kvstore.Client
	desc   *prometheus.Desc
	logger log.Logger
}

func (c *redisClient) Collect(namespace string, ch chan<- prometheus.Metric) error {
	if c.desc == nil {
		c.desc = newDescOfInstnaceInfo(namespace, "acs_kvstore", []string{"regionId", "instanceId", "name", "connectionDomain", "address", "status"})
	}
	req := r_kvstore.CreateDescribeInstancesRequest()
	req.PageSize = requests.NewInteger(pageSize)
	resultChan := make(chan *result, 1<<10)
	var totalCount int

	go func() {
		defer close(resultChan)
		for hasNextPage, pageNum := true, 1; hasNextPage != false; pageNum++ {
			req.PageNumber = requests.NewInteger(pageNum)
			response, err := c.DescribeInstances(req)
			if err != nil {
				resultChan <- &result{err: err}
				return
			}
			totalCount += len(response.Instances.KVStoreInstance)
			if totalCount >= response.TotalCount {
				hasNextPage = false
			}
			for i := range response.Instances.KVStoreInstance {
				resultChan <- &result{v: response.Instances.KVStoreInstance[i]}
			}
		}
	}()

	for res := range resultChan {
		if res.err != nil {
			return res.err
		}
		ins := res.v.(r_kvstore.KVStoreInstanceInDescribeInstances)
		ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, 1.0,
			ins.RegionId, ins.InstanceId, ins.InstanceName, ins.ConnectionDomain, ins.PrivateIp, ins.InstanceStatus)
	}
	return nil
}

func init() {
	register("acs_kvstore", func(s1, s2, s3 string, l log.Logger) (Collector, error) {
		client, err := r_kvstore.NewClientWithAccessKey(s1, s2, s3)
		if err != nil {
			return nil, err
		}
		return &redisClient{Client: client, logger: l}, nil
	})
}

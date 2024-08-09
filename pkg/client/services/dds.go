package services

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/dds"
	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
)

type ddsClient struct {
	*dds.Client
	desc   *prometheus.Desc
	logger log.Logger
}

func (c *ddsClient) Collect(namespace string, ch chan<- prometheus.Metric) error {
	if c.desc == nil {
		c.desc = newDescOfInstnaceInfo(namespace, "mongo", []string{"regionId", "instanceId", "dbType", "desc", "status"})
	}

	req := dds.CreateDescribeDBInstancesRequest()
	req.PageSize = requests.NewInteger(pageSize)
	resultChan := make(chan *result, 1<<10)
	go func() {
		defer close(resultChan)
		for hasNextPage, pageNum := true, 1; hasNextPage != false; pageNum++ {
			req.PageNumber = requests.NewInteger(pageNum)
			response, err := c.DescribeDBInstances(req)
			if err != nil {
				resultChan <- &result{err: err}
				return
			}
			if len(response.DBInstances.DBInstance) < pageSize {
				hasNextPage = false
			}
			for i := range response.DBInstances.DBInstance {
				resultChan <- &result{v: response.DBInstances.DBInstance[i]}
			}
		}
	}()
	for res := range resultChan {
		if res.err != nil {
			return res.err
		}
		ins := res.v.(dds.DBInstance)
		ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, 1.0,
			ins.RegionId, ins.DBInstanceId, ins.DBInstanceType, ins.DBInstanceDescription, ins.DBInstanceStatus)
	}
	return nil
}

func init() {
	register("mongo", func(s1, s2, s3 string, l log.Logger) (Collector, error) {
		client, err := dds.NewClientWithAccessKey(s1, s2, s3)
		if err != nil {
			return nil, err
		}
		return &ddsClient{Client: client, logger: l}, nil
	})
}

package services

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/rds"
	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
)

// Client wrap client
type rdsClient struct {
	*rds.Client
	desc   *prometheus.Desc
	logger log.Logger
}

func (c *rdsClient) Collect(namespace string, ch chan<- prometheus.Metric) error {
	if c.desc == nil {
		c.desc = newDescOfInstnaceInfo(namespace, "acs_rds_dashboard",
			[]string{"regionId", "dbInstanceId", "name", "dbType", "desc", "status"})
	}
	req := rds.CreateDescribeDBInstancesRequest()
	req.PageSize = requests.NewInteger(pageSize)
	resultChan := make(chan *result, 1<<10)
	var totalCount int

	go func() {
		defer close(resultChan)
		for hasNextPage, pageNum := true, 1; hasNextPage != false; pageNum++ {
			req.PageNumber = requests.NewInteger(pageNum)
			response, err := c.DescribeDBInstances(req)
			if err != nil {
				resultChan <- &result{err: err}
				return
			}
			totalCount += len(response.Items.DBInstance)
			if totalCount >= response.TotalRecordCount {
				hasNextPage = false
			}
			for i := range response.Items.DBInstance {
				resultChan <- &result{v: response.Items.DBInstance[i]}
			}
		}
	}()

	for res := range resultChan {
		if res.err != nil {
			return res.err
		}
		ins := res.v.(rds.DBInstance)
		ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, 1.0,
			ins.RegionId, ins.DBInstanceId, ins.DBInstanceName, ins.DBInstanceType, ins.DBInstanceDescription, ins.DBInstanceStatus)
	}
	return nil
}

func init() {
	register("acs_rds_dashboard", func(s1, s2, s3 string, l log.Logger) (Collector, error) {
		client, err := rds.NewClientWithAccessKey(s1, s2, s3)
		if err != nil {
			return nil, err
		}
		return &rdsClient{Client: client, logger: l}, nil
	})
}

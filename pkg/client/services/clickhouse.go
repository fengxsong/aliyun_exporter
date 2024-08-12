package services

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/clickhouse"
	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
)

type ckClient struct {
	*clickhouse.Client
	desc   *prometheus.Desc
	logger log.Logger
}

func (c *ckClient) Collect(namespace string, ch chan<- prometheus.Metric) error {
	if c.desc == nil {
		c.desc = newDescOfInstnaceInfo(namespace, "acs_clickhouse", []string{"regionId", "dbclusterId", "type", "status", "desc"})
	}

	req := clickhouse.CreateDescribeDBClustersRequest()
	req.PageSize = requests.NewInteger(pageSize)
	resultChan := make(chan *result, 1<<10)
	var totalCount int

	go func() {
		defer close(resultChan)
		for hasNextPage, pageNum := true, 1; hasNextPage != false; pageNum++ {
			req.PageNumber = requests.NewInteger(pageNum)
			req.PageSize = requests.NewInteger(pageSize)
			response, err := c.DescribeDBClusters(req)
			if err != nil {
				resultChan <- &result{err: err}
				return
			}
			totalCount += len(response.DBClusters.DBCluster)
			if totalCount >= response.TotalCount {
				hasNextPage = false
			}
			for i := range response.DBClusters.DBCluster {
				resultChan <- &result{v: response.DBClusters.DBCluster[i]}
			}
		}
	}()
	for res := range resultChan {
		if res.err != nil {
			return res.err
		}
		ins := res.v.(clickhouse.DBCluster)
		ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, 1.0,
			ins.RegionId, ins.DBClusterId, ins.DBClusterType, ins.DBClusterStatus, ins.DBClusterDescription)
	}
	return nil
}

func init() {
	register("acs_clickhouse", func(s1, s2, s3 string, l log.Logger) (Collector, error) {
		client, err := clickhouse.NewClientWithAccessKey(s1, s2, s3)
		if err != nil {
			return nil, err
		}
		return &ckClient{Client: client, logger: l}, nil
	})
}

package services

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/elasticsearch"
	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
)

type esClient struct {
	*elasticsearch.Client
	desc   *prometheus.Desc
	logger log.Logger
}

func (c *esClient) Collect(namespace string, ch chan<- prometheus.Metric) error {
	if c.desc == nil {
		c.desc = newDescOfInstnaceInfo(namespace, "acs_elasticsearch", []string{"instanceId", "desc", "status"})
	}

	req := elasticsearch.CreateListInstanceRequest()
	req.Size = requests.NewInteger(pageSize)
	resultChan := make(chan *result, 1<<10)

	go func() {
		defer close(resultChan)
		for hasNextPage, pageNum := true, 1; hasNextPage != false; pageNum++ {
			req.Page = requests.NewInteger(pageNum)
			response, err := c.ListInstance(req)
			if err != nil {
				resultChan <- &result{err: err}
				return
			}
			if len(response.Result) < pageSize && len(response.Result) == 0 {
				hasNextPage = false
			}
			for i := range response.Result {
				resultChan <- &result{v: response.Result[i]}
			}
		}
	}()

	for res := range resultChan {
		if res.err != nil {
			return res.err
		}
		ins := res.v.(elasticsearch.Instance)
		ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, 1.0,
			ins.InstanceId, ins.Description, ins.Status)
	}
	return nil
}

func init() {
	register("acs_elasticsearch", func(s1, s2, s3 string, l log.Logger) (Collector, error) {
		client, err := elasticsearch.NewClientWithAccessKey(s1, s2, s3)
		if err != nil {
			return nil, err
		}
		return &esClient{Client: client, logger: l}, nil
	})
}

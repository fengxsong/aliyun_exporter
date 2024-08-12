package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/cms"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/fengxsong/aliyun_exporter/pkg/client/services"
	"github.com/fengxsong/aliyun_exporter/pkg/config"
)

var ignores = map[string]struct{}{
	"timestamp": {},
	"Maximum":   {},
	"Minimum":   {},
	"Average":   {},
}

// Datapoint datapoint
type Datapoint map[string]interface{}

// Get return value for measure
func (d Datapoint) Get(measure string) float64 {
	v, ok := d[measure]
	if !ok {
		return 0
	}
	return v.(float64)
}

// Labels return labels that not in ignores
func (d Datapoint) Labels() []string {
	labels := make([]string, 0)
	for k := range d {
		if _, ok := ignores[k]; !ok {
			labels = append(labels, k)
		}
	}
	sort.Strings(labels)
	return labels
}

// Values return values for lables
func (d Datapoint) Values(labels ...string) []string {
	values := make([]string, 0, len(labels))
	for i := range labels {
		values = append(values, fmt.Sprintf("%s", d[labels[i]]))
	}
	return values
}

// MetricClient wrap cms.client
type MetricClient struct {
	cms    *cms.Client
	logger log.Logger
}

// NewMetricClient create metric Client
func NewMetricClient(ak, secret, region string, rt http.RoundTripper, logger log.Logger) (*MetricClient, error) {
	cmsClient, err := cms.NewClientWithAccessKey(region, ak, secret)
	if err != nil {
		return nil, err
	}
	cmsClient.SetTransport(rt)
	if logger == nil {
		logger = log.NewNopLogger()
	}
	return &MetricClient{cmsClient, logger}, nil
}

// retrive get datapoints for metric
// TODO: can we using BatchExport function instead? seems like this function returned a time-range metrics series
func (c *MetricClient) retrive(sub string, name string, period string) ([]Datapoint, error) {
	var ret []Datapoint

	var (
		firstRun  = true
		nextToken string
	)

	for {
		if !firstRun && nextToken == "" {
			break
		}
		req := cms.CreateDescribeMetricLastRequest()
		req.Namespace = sub
		req.MetricName = name
		req.Period = period
		req.NextToken = nextToken
		resp, err := c.cms.DescribeMetricLast(req)
		if err != nil {
			return nil, err
		}

		var datapoints []Datapoint
		if err = json.Unmarshal([]byte(resp.Datapoints), &datapoints); err != nil {
			// some unexpected error
			level.Error(c.logger).Log("content", resp.GetHttpContentString(), "error", err)
			return nil, err
		}
		ret = append(ret, datapoints...)
		nextToken = resp.NextToken
		firstRun = false
	}

	return ret, nil
}

// Collect do collect metrics into channel
func (c *MetricClient) Collect(namespace string, sub string, m *config.Metric, ch chan<- prometheus.Metric) error {
	if m.Name == "" {
		level.Warn(c.logger).Log("msg", "metric name must been set")
		return nil
	}
	datapoints, err := c.retrive(sub, m.Name, m.Period)
	if err != nil {
		return err
	}
	for _, dp := range datapoints {
		val := dp.Get(m.Measure)
		ch <- prometheus.MustNewConstMetric(
			m.Desc(namespace, sub, dp.Labels()...),
			prometheus.GaugeValue,
			val,
			dp.Values(m.Dimensions...)...,
		)
	}
	return nil
}

// TODO: is there any convenient way to list all those namespaces?
func (c *MetricClient) ListingNamespace() ([]string, error) {
	resources, err := c.describeMetricMetaListWithNamespace("")
	if err != nil {
		return nil, err
	}
	m := make(map[string]struct{})
	var ret []string
	for _, res := range resources {
		if _, ok := m[res.Namespace]; !ok {
			m[res.Namespace] = struct{}{}
			ret = append(ret, res.Namespace)
		}
	}
	sort.Strings(ret)
	return ret, nil
}

func (c *MetricClient) describeMetricMetaListWithNamespace(namespace string) ([]cms.Resource, error) {
	var ret []cms.Resource
	pageNumber := 1
	for {
		req := cms.CreateDescribeMetricMetaListRequest()
		if namespace != "" {
			req.Namespace = namespace
		}
		req.PageSize = requests.NewInteger(1 << 8)
		req.PageNumber = requests.NewInteger(pageNumber)
		resp, err := c.cms.DescribeMetricMetaList(req)
		if err != nil {
			return nil, err
		}
		level.Debug(c.logger).Log("response", resp.GetHttpContentString())
		totalCount, err := strconv.Atoi(resp.TotalCount)
		if err != nil {
			return nil, err
		}
		ret = append(ret, resp.Resources.Resource...)
		pageNumber++
		if len(ret) >= totalCount {
			break
		}
	}
	return ret, nil
}

func (c *MetricClient) describeMetricMetaList(includes ...string) ([]cms.Resource, error) {
	if len(includes) == 0 {
		return c.describeMetricMetaListWithNamespace("")
	}

	type metricMetaResult struct {
		resources []cms.Resource
		err       error
	}

	ch := make(chan *metricMetaResult, len(includes))
	wg := &sync.WaitGroup{}
	for i := range includes {
		wg.Add(1)
		go func(namespace string) {
			defer wg.Done()
			res, err := c.describeMetricMetaListWithNamespace(namespace)
			if err != nil {
				close(ch)
				ch <- &metricMetaResult{err: err}
				return
			}
			ch <- &metricMetaResult{resources: res}
		}(includes[i])
	}
	wg.Wait()
	close(ch)

	var ret []cms.Resource
	for res := range ch {
		if res.err != nil {
			return nil, res.err
		}
		ret = append(ret, res.resources...)
	}
	return ret, nil
}

// GenerateExampleConfig create example config
func (c *MetricClient) GenerateExampleConfig(ak, sk, region string, includes ...string) (*config.Config, error) {
	metas, err := c.describeMetricMetaList(includes...)
	if err != nil {
		return nil, err
	}
	cfg := &config.Config{
		AccessKey:       "<changeme>",
		AccessKeySecret: "<changeme>",
		Region:          region,
		Metrics:         make(map[string][]*config.Metric),
	}

	builtin := services.Names()
	if len(includes) == 0 {
		level.Info(c.logger).Log("msg", "no collectors specified, using builtin instance info collectors will be used", "collectors", strings.Join(builtin, ", "))
		cfg.InstanceTypes = builtin
	} else {
		tmp := make(map[string]struct{})
		for i := range builtin {
			tmp[builtin[i]] = struct{}{}
		}
		var unsupported []string
		for i := range includes {
			if _, ok := tmp[includes[i]]; !ok {
				unsupported = append(unsupported, includes[i])
				continue
			}
			cfg.InstanceTypes = append(cfg.InstanceTypes, includes[i])
		}
		if len(unsupported) > 0 {
			level.Warn(c.logger).
				Log("msg", fmt.Sprintf("unsupported instance info collectors %s, implement by your own OR skip scraping instance info for those types", unsupported))
		}
	}

	for _, res := range metas {
		if _, ok := cfg.Metrics[res.Namespace]; !ok {
			cfg.Metrics[res.Namespace] = make([]*config.Metric, 0)
		}
		cfg.Metrics[res.Namespace] = append(cfg.Metrics[res.Namespace], &config.Metric{
			Name:        res.MetricName,
			Period:      res.Periods,
			Description: res.Description,
			Dimensions:  strings.Split(res.Dimensions, ","),
			Unit:        res.Unit,
			Measure:     res.Statistics,
		})
	}
	return cfg, nil
}

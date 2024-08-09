# Aliyun Exporter

a prometheus exporter for Aliyun CloudMonitor service. Written in Golang, inspired by [aliyun_exporter](https://github.com/aylei/aliyun_exporter).

## Develop

```bash
git clone https://github.com/fengxsong/aliyun_exporter
cd aliyun_exporter
make tidy
```

## Usage

```bash
make bin
# generate example of config
./build/_output/bin/aliyun_exporter generate-example-config --accesskey xxxx ----accesskeysecret xxxx
# run http metrics handler
./build/_output/bin/aliyun_exporter serve [--config=/path/of/config]
```

### Create a prometheus scrape job

```yaml
- job_name: 'aliyun_exporter'
  scrape_interval: 60s
  scrape_timeout: 60s
  static_configs:
  - targets: ['aliyun_exporter:9527']
    labels:
      account_name: xxxx
      provider: aliyun # or aliyun_jst
```

### Prometheus rules sample

[sample file](https://../deploy/rules.yaml)

## Limitation

- an exporter instance can only scrape metrics from single region

## TODO

- grafana dashboard

## Ref

- [Metrics listing](https://www.alibabacloud.com/help/en/cms/support/appendix-1-metrics)
- [DescribeMetricMetaList](https://www.alibabacloud.com/help/en/cms/developer-reference/api-cms-2019-01-01-describemetricmetalist?spm=a2c63.p38356.0.0.12e23344MoHCLk)
- [DescribeMetricLast](https://www.alibabacloud.com/help/en/cms/developer-reference/api-cms-2019-01-01-describemetriclast?spm=a2c63.p38356.0.0.7e7d5b35Ml1iqb)
- [IAM access](https://help.aliyun.com/zh/ram/developer-reference/aliyuncloudmonitorreadonlyaccess?spm=a2c4g.11186623.0.0.4eafd1eaFYcOoS)
- [SDK](https://github.com/aliyun/alibaba-cloud-sdk-go)

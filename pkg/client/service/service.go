package service

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"
)

// Collector instance collector
// create(?) and collect metric into channel
type Collector interface {
	Collect(namespace string, ch chan<- prometheus.Metric)
}

// Services return registered service name
func Services() []string {
	names := make([]string, 0, len(collectorFunc))
	for k := range collectorFunc {
		names = append(names, k)
	}
	return names
}

// CollectorFunc return registered func map
func CollectorFunc() map[string]func(string, string, string, http.RoundTripper, log.Logger) (Collector, error) {
	return collectorFunc
}

var lock = &sync.Mutex{}
var collectorFunc = map[string]func(string, string, string, http.RoundTripper, log.Logger) (Collector, error){}

// Register register new collector function
func Register(name string, fn func(string, string, string, http.RoundTripper, log.Logger) (Collector, error)) {
	lock.Lock()
	defer lock.Unlock()
	if _, ok := collectorFunc[name]; ok {
		panic("function already registered")
	}
	collectorFunc[name] = fn
}

// NewInstanceClientDesc ..
func NewInstanceClientDesc(namespace string, sub string, variableLabels []string) *prometheus.Desc {
	return prometheus.NewDesc(
		prometheus.BuildFQName(namespace, sub, "instance_info"),
		fmt.Sprintf("%s instance info", sub),
		variableLabels, nil,
	)
}

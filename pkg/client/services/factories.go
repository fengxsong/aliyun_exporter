package services

import (
	"fmt"
	"sort"
	"sync"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
)

type Collector interface {
	Collect(string, chan<- prometheus.Metric) error
}

type NewCollectorFunc func(string, string, string, log.Logger) (Collector, error)

// Names return all registered service collector name
func Names() []string {
	names := make([]string, 0, len(factories))
	for k := range factories {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// All return all registered newCollector functions map
func All() map[string]NewCollectorFunc {
	return factories
}

var lock = &sync.Mutex{}
var factories = map[string]NewCollectorFunc{}

// Register register new collector function
func register(name string, fn NewCollectorFunc) {
	lock.Lock()
	defer lock.Unlock()
	if _, ok := factories[name]; ok {
		panic("named collector function already registered")
	}
	factories[name] = fn
}

// the maximum page size is 100, otherwise it'll return InvalidParameter error
const pageSize = 100

func newDescOfInstnaceInfo(namespace string, sub string, variableLabels []string) *prometheus.Desc {
	return prometheus.NewDesc(
		prometheus.BuildFQName(namespace, sub, "instance_info"),
		fmt.Sprintf("Information of instance %s", sub),
		variableLabels, nil,
	)
}

type result struct {
	v   any
	err error
}

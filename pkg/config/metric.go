package config

import (
	"fmt"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// Metric meta
type Metric struct {
	Name        string   `yaml:"name"`
	Alias       string   `yaml:"alias,omitempty"`
	Period      string   `yaml:"period,omitempty"`
	Description string   `yaml:"desc,omitempty"`
	Dimensions  []string `yaml:"dimensions,omitempty"`
	Unit        string   `yaml:"unit,omitempty"`
	Measure     string   `yaml:"measure,omitempty"`
	Format      bool     `yaml:"format,omitempty"`

	once sync.Once
	desc *prometheus.Desc
}

// setdefault options
func (m *Metric) setDefaults() {
	if m.Period == "" {
		m.Period = "60"
	}
	if m.Description == "" {
		m.Description = m.Name
	}
	// Do some fallback in case someone runs this exporter directly
	// without modifying the example configuration
	periods := strings.Split(m.Period, ",")
	m.Period = periods[0]
	measures := strings.Split(m.Measure, ",")
	m.Measure = measures[0]
	switch m.Measure {
	case "Maximum", "Minimum", "Average", "Value":
	default:
		m.Measure = "Average"
	}
	m.Description = fmt.Sprintf("%s unit:%s measure:%s", m.Description, m.Unit, m.Measure)
}

var formalizeMetricName = strings.NewReplacer(".", "_", "-", "_").Replace

// String name of metric
func (m *Metric) String() string {
	if m.Alias != "" {
		return m.Alias
	}
	if m.Format {
		return strings.Join([]string{m.Name, formatUnit(m.Unit)}, "_")
	}
	return formalizeMetricName(m.Name)
}

// Desc to prometheus desc
func (m *Metric) Desc(namespace, sub string, dimensions ...string) *prometheus.Desc {
	if len(m.Dimensions) == 0 {
		m.Dimensions = dimensions
	}
	// TODO: if length of dimemsions is changed
	m.once.Do(func() {
		m.desc = prometheus.NewDesc(
			prometheus.BuildFQName(namespace, sub, m.String()),
			m.Description,
			m.Dimensions,
			nil,
		)
	})
	return m.desc
}

var durationUnitMapping = map[string]string{
	"s": "second",
	"m": "minute",
	"h": "hour",
	"d": "day",
}

func formatUnit(s string) string {
	s = strings.TrimSpace(s)
	if s == "%" {
		return "percent"
	}

	if strings.IndexAny(s, "/") > 0 {
		fields := strings.Split(s, "/")
		if len(fields) == 2 {
			if v, ok := durationUnitMapping[fields[1]]; ok {
				return strings.ToLower(strings.Join([]string{fields[0], "per", v}, "_"))
			}
		}
	}
	return strings.ToLower(s)
}

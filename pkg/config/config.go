package config

import (
	"errors"
	"io/ioutil"
	"os"

	"sigs.k8s.io/yaml"
)

// Config exporter config
type Config struct {
	AccessKey       string `json:"accessKey"`
	AccessKeySecret string `json:"accessKeySecret"`
	Region          string `json:"region"`
	// todo: add extra labels
	Labels        map[string]string    `json:"labels,omitempty"`
	Metrics       map[string][]*Metric `json:"metrics"` // mapping for namespace and metrics
	InstanceInfos []string             `json:"instanceInfos"`
}

func (c *Config) setDefaults() {
	if c.Region == "" {
		c.Region = "cn-hangzhou"
	}
	for _, metrics := range c.Metrics {
		for i := range metrics {
			metrics[i].setDefaults()
		}
	}
}

// Parse parse config from file
func Parse(path string) (*Config, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err = yaml.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	ak := getEnvOrDefault("ACCESS_KEY", cfg.AccessKey)
	secret := getEnvOrDefault("ACCESS_KEY_SECRET", cfg.AccessKeySecret)
	if len(ak) == 0 || len(secret) == 0 {
		return nil, errors.New("credentials not provide")
	}
	cfg.setDefaults()
	return &cfg, nil
}

func getEnvOrDefault(key string, defaultVal string) string {
	val := os.Getenv(key)
	if val != "" {
		return val
	}
	return defaultVal
}

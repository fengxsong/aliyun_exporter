package config

import (
	"errors"
	"os"

	"gopkg.in/yaml.v2"
)

// Config exporter config
type Config struct {
	AccessKey       string               `yaml:"access_key"`
	AccessKeySecret string               `yaml:"access_key_secret"`
	Region          string               `yaml:"region"`
	Labels          map[string]string    `yaml:"labels,omitempty"` // todo: add extra labels
	Metrics         map[string][]*Metric `yaml:"metrics"`          // mapping for namespace and metrics
	InstanceTypes   []string             `yaml:"instance_types"`
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
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err = yaml.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	ak := getEnvOrDefault("ALIBABA_CLOUD_ACCESS_KEY", cfg.AccessKey)
	secret := getEnvOrDefault("ALIBABA_CLOUD_ACCESS_KEY_SECRET", cfg.AccessKeySecret)
	if len(ak) == 0 || len(secret) == 0 {
		return nil, errors.New("credentials not provide")
	}
	cfg.setDefaults()
	return &cfg, nil
}

func getEnvOrDefault(key string, defaultVal string) string {
	val, ok := os.LookupEnv(key)
	if ok {
		return val
	}
	return defaultVal
}

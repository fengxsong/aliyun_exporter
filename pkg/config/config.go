package config

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

// Config exporter config
type Config struct {
	AccessKey       string               `yaml:"access_key"`
	AccessKeySecret string               `yaml:"access_key_secret"`
	Region          string               `yaml:"region"`
	Labels          map[string]string    `yaml:"labels,omitempty"` // todo: add extra labels
	Metrics         map[string][]*Metric `yaml:"metrics"`          // mapping for namespace and metrics
	InstanceInfo    *InstanceInfo        `yaml:"instance_info"`    // enable scraping instance infos for label join
}

type InstanceInfo struct {
	Types   []string `yaml:"types"`
	Regions []string `yaml:"regions"`
}

func (c *Config) validateAndSetDefaults() error {
	if c.Region == "" {
		c.Region = "cn-hangzhou"
	}
	ak := getEnvOrDefault("ALIBABA_CLOUD_ACCESS_KEY", c.AccessKey)
	secret := getEnvOrDefault("ALIBABA_CLOUD_ACCESS_KEY_SECRET", c.AccessKeySecret)
	if len(ak) == 0 || len(secret) == 0 {
		return errors.New("credentials not provide")
	}
	c.AccessKey = ak
	c.AccessKeySecret = secret
	if c.InstanceInfo != nil && len(c.InstanceInfo.Regions) == 0 {
		c.InstanceInfo.Regions = []string{c.Region}
	}
	for _, metrics := range c.Metrics {
		for i := range metrics {
			metrics[i].setDefaults()
		}
	}
	return nil
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
	if err = cfg.validateAndSetDefaults(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func getEnvOrDefault(key string, defaultVal string) string {
	val, ok := os.LookupEnv(key)
	if ok {
		return val
	}
	return defaultVal
}

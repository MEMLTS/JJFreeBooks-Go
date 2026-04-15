package config

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Token       string    `yaml:"token"`
	Cron        string    `yaml:"cron"`
	Intervals   Intervals `yaml:"intervals"`
	NovelFilter []string  `yaml:"novel_filter"`
}

type Intervals struct {
	Chapter    int `yaml:"chapter"`
	Book       int `yaml:"book"`
	LegacyBook int `yaml:"novel"`
}

func defaultConfig() Config {
	return Config{
		Token:       "",
		Cron:        "0 0 * * *",
		NovelFilter: []string{"all"},
		Intervals: Intervals{
			Book:    1000,
			Chapter: 500,
		},
	}
}

func LoadConfig() (Config, error) {
	fileName := "config.yaml"
	_, err := os.Stat(fileName)
	if os.IsNotExist(err) {
		config := defaultConfig()
		data, err := yaml.Marshal(config)
		if err != nil {
			return Config{}, err
		}
		err = os.WriteFile(fileName, data, 0644)
		if err != nil {
			return Config{}, err
		}
	}

	data, err := os.ReadFile(fileName)
	if err != nil {
		return Config{}, err
	}
	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return Config{}, err
	}

	config.normalize()
	return config, config.Validate()
}

func (c *Config) normalize() {
	if strings.TrimSpace(c.Cron) == "" {
		c.Cron = "0 0 * * *"
	}
	if c.Intervals.Book <= 0 {
		c.Intervals.Book = c.Intervals.LegacyBook
	}
	if c.Intervals.Book <= 0 {
		c.Intervals.Book = 1000
	}
	if c.Intervals.Chapter <= 0 {
		c.Intervals.Chapter = 500
	}
	if len(c.NovelFilter) == 0 {
		c.NovelFilter = []string{"all"}
	}
	c.Token = strings.TrimSpace(c.Token)
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.Cron) == "" {
		c.Cron = "0 0 * * *"
	}
	return nil
}

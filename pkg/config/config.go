// Package config provides functions for parsing a day-two config file.
package config

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// ChartList is the struct to hold a config file.
type ChartList struct {
	Charts map[string]Chart `yaml:"charts"`
}

// Chart is a single struct definition in the config file.
type Chart struct {
	Namespace    string   `yaml:"namespace"`
	Repo         string   `yaml:"repo"`
	Chart        string   `yaml:"chart"`
	ValuesPath   string   `yaml:"valuesPath"`
	Dependencies []string `yaml:"depends"`
}

// LoadConfig will pull in a config file at the specified path and turn it
// into a ChartList.
func LoadConfig(configPath string) (ChartList, error) {
	var retChartList ChartList

	configFile, err := ioutil.ReadFile(configPath)
	if err != nil {
		return retChartList, err
	}

	err = yaml.Unmarshal(configFile, &retChartList)
	if err != nil {
		return retChartList, err
	}

	return retChartList, nil
}

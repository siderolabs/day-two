package config

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type ChartList struct {
	Charts map[string]Chart `yaml:"charts"`
}

type Chart struct {
	Namespace    string   `yaml:"namespace"`
	Repo         string   `yaml:"repo"`
	Chart        string   `yaml:"chart"`
	ValuesPath   string   `yaml:"valuesPath"`
	Dependencies []string `yaml:"depends"`
}

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

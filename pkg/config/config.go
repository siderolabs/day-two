// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package config manages the configuration for d2ctl
package config

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// ChartList stores the charts deployed.
type ChartList struct {
	Charts map[string]Chart `yaml:"charts"`
}

// Chart holds the Helm chart information to deploy.
type Chart struct {
	Namespace        string   `yaml:"namespace"`
	PodSecurityLevel string   `yaml:"podSecurityLevel"`
	Repo             string   `yaml:"repo"`
	Chart            string   `yaml:"chart"`
	ValuesPath       string   `yaml:"valuesPath"`
	Dependencies     []string `yaml:"depends"`
}

// LoadConfig loads the configuration yaml file from a path.
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

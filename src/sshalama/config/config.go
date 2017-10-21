// Sshalama
//
// Copyright 2016-2017 Dolf Schimmel
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package config

import (
	"fmt"

	"gopkg.in/gcfg.v1"
)

type Config struct {
	General GeneralConfig `gcfg:"sshalama"`
	Listen  map[string]*struct {
		Bind          string
		ProxyProtocol bool `gcfg:"proxy-protocol"`
	}
}

type GeneralConfig struct {
	Include []string
}

func NewConfig(loadFromFile string) (*Config, error) {
	c := getDefaultConfig()

	if loadFromFile != "" {
		if err := c.loadFromFile(loadFromFile); err != nil {
			return nil, err
		}
	}

	return c, nil
}

func getDefaultConfig() *Config {
	return &Config{
		General: GeneralConfig{
			Include: make([]string, 0),
		},
	}
}

func (c *Config) validate() error {
	return nil
}

func (c *Config) loadFromFile(configPath string) error {
	err := gcfg.ReadFileInto(c, configPath)
	if err != nil {
		return fmt.Errorf("could not parse %s: %s", configPath, err.Error())
	}

	for i, include := range c.General.Include {
		if include == "" {
			continue
		}
		c.General.Include[i] = "" // Prevent infinite recursion

		if err := c.loadFromFile(include); err != nil {
			return err
		}
	}

	if err = c.validate(); err != nil {
		return fmt.Errorf("could not parse %s: %s", configPath, err.Error())
	}

	return nil
}

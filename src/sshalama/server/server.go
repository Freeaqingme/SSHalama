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
package server

import (
	"sshalama/config"
)

type Server struct {
}

func Start(configPath string) error {
	s, config, err := newServer(configPath)
	if err != nil {
		return err
	}

	go s.run(config)

	return nil
}

func newServer(configPath string) (*Server, *config.Config, error) {
	config, err := config.NewConfig(configPath)
	if err != nil {
		return nil, nil, err
	}

	s := &Server{}
	return s, config, nil
}

func (s *Server) run(config *config.Config) error {
	for _, listen := range config.Listen {
		if err := s.listen(listen.Bind, listen.ProxyProtocol); err != nil {
			return err
		}
	}

	return nil
}

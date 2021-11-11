/*
Copyright 2021 The Kuda Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package webhook

import (
	"io/ioutil"

	"k8s.io/apimachinery/pkg/util/yaml"
)

// Config defines fields for webhook.
type Config struct {
	RuntimeImage      string `yaml:"runtimeImage"`
	HostPath          string `yaml:"hostPath"`
	DataPathPrefix    string `yaml:"dataPathPrefix"`
	EnableAffinity    bool   `yaml:"enableAffinity"`
	RuntimeServerPort uint   `yaml:"runtimeServerPort"`
}

// LoadConfig returns config from the file.
func LoadConfig(file string) (*Config, error) {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

package config

import (
	"encoding/json"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/hashicorp/hcl"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"strings"
)

type Format string

const (
	JSONFormat Format = "json"
	TOMLFormat Format = "toml"
	YAMLFormat Format = "yaml"
	HCLFormat  Format = "hcl"
)

func NewDefault() *Config {
	return &Config{}
}

func NewFromFile(filename string) (*Config, error) {
	var format Format
	if strings.HasSuffix(filename, ".json") {
		format = JSONFormat
	} else if strings.HasSuffix(filename, ".toml") {
		format = TOMLFormat
	} else if strings.HasSuffix(filename, ".yaml") {
		format = YAMLFormat
	} else if strings.HasSuffix(filename, ".hcl") {
		format = HCLFormat
	} else {
		return nil, fmt.Errorf("Unrecognized file format for config file: %v", filename)
	}
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return NewFromBytes(bytes, format)
}

func NewFromBytes(bytes []byte, format Format) (*Config, error) {
	switch format {
	case JSONFormat:
		return newFromJSONBytes(bytes)
	case TOMLFormat:
		return newFromTOMLBytes(bytes)
	case YAMLFormat:
		return newFromYAMLBytes(bytes)
	case HCLFormat:
		return newFromHCLBytes(bytes)
	default:
		return nil, fmt.Errorf("Unrecognized format: %v", format)
	}
}

func newFromJSONBytes(bytes []byte) (*Config, error) {
	conf := new(Config)
	if err := json.Unmarshal(bytes, conf); err != nil {
		return nil, err
	}
	return conf, nil
}

func newFromTOMLBytes(bytes []byte) (*Config, error) {
	conf := new(Config)
	if err := toml.Unmarshal(bytes, conf); err != nil {
		return nil, err
	}
	return conf, nil
}

func newFromYAMLBytes(bytes []byte) (*Config, error) {
	conf := new(Config)
	if err := yaml.Unmarshal(bytes, conf); err != nil {
		return nil, err
	}
	return conf, nil
}

func newFromHCLBytes(bytes []byte) (*Config, error) {
	conf := new(Config)
	if err := hcl.Decode(conf, string(bytes)); err != nil {
		return nil, err
	}
	return conf, nil
}

func (c *Config) ToJSON(pretty bool) ([]byte, error) {
	if pretty {
		return json.MarshalIndent(c, "", "  ")
	}
	return json.Marshal(c)
}

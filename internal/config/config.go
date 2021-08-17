package config

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type NatsConfig struct {
	Url          string `yaml:"url"`
	Username     string `yaml:"username",omitempty`
	PasswordFile string `yaml:"passwordFile",omitempty`
}

type MidiConfig struct {
	Input         string `yaml:"input"`
	Output        string `yaml:"output"`
	Channel       uint   `yaml:"channel"`
	MaxInputValue uint   `yaml:"maxInputValue"`
}

type PulseaudioConfig struct {
	Targets map[string]struct {
		Type string `yaml:"type"`
		Name string `yaml:"name"`
	} `yaml:"targets"`
}

type Action struct {
	Type   string                 `yaml:"type"`
	Config map[string]interface{} `yaml:"config"`
}

type Config struct {
	Nats       NatsConfig       `yaml:"nats"`
	Midi       MidiConfig       `yaml:"midi"`
	Pulseaudio PulseaudioConfig `yaml:"pulseaudio"`
	Action     []Action         `yaml:"actions"`
}

func Read(filename string) (*Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	config := &Config{}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

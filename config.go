package main

import (
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

type ServerConfig struct {
	Address string `yaml:"address"`
	Tlsname string `yaml:"tls"`
}

type ClientConfig struct {
	Address string `yaml:"address"`
	Tlsname string `yaml:"tls"`
}

type ChannelConfig struct {
	Protocol string `yaml:"protocol"`
	Local    string `yaml:"local"`
	Remote   string `yaml:"remote"`
}

type TlsConfig struct {
	Name string `yaml:"name"`
	Cert string `yaml:"cert"`
	Key  string `yaml:"key"`
	CA   string `yaml:"ca"`
}

type GlobalConfig struct {
	Server  ServerConfig    `yaml:"server"`
	Client  ClientConfig    `yaml:"client"`
	Channel []ChannelConfig `yaml:"channel"`
	Tls     []TlsConfig     `yaml:"tls"`
}

var globalconfig *GlobalConfig

func LoadConfig(filename string) error {

	body, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	config := new(GlobalConfig)
	config.Channel = make([]ChannelConfig, 0)
	config.Tls = make([]TlsConfig, 0)

	err = yaml.Unmarshal(body, config)
	if err != nil {
		return err
	}

	log.Printf("%v", config)

	globalconfig = config
	return nil
}

func ServerCfgGet() ServerConfig {
	return globalconfig.Server
}

func ClientCfgGet() ClientConfig {
	return globalconfig.Client
}

func ChannelCfgGet() []ChannelConfig {
	return globalconfig.Channel
}

func TlsCfgGet(name string) *TlsConfig {
	for _, v := range globalconfig.Tls {
		if v.Name == name {
			return &v
		}
	}
	return nil
}

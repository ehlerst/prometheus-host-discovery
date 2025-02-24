package main

import (
	"github.com/golang/glog"
	"gopkg.in/yaml.v3"
	"io"
	"os"
)

type SDConfig struct {
	Networks    []Networks `yaml:"networks"`
	Concurrency int        `yaml:"concurrency"`
	Filesdpath  string     `yaml:"filesdpath"`
	Port        []int      `yaml:"port"`
	Timeout     int        `yaml:"timeout"`
}

type Networks struct {
	Labels  []struct {
		NetworkName string `yaml:"networkname"`
	} `yaml:"labels"`
	Network string   `yaml:"network"`
}

func newSDConfig(filename string) (*SDConfig,error) {
	var hosts SDConfig
	yamlFile, err := os.Open(filename)
	if err != nil {
		glog.Error("cant open: ",filename," ",err)
		return nil, err
	}
	byteValue, _ := io.ReadAll(yamlFile)
	err = yaml.Unmarshal(byteValue,&hosts)
	if err != nil {
		glog.Error("cant parse: ",filename," ",err)
		return nil,err
	}
	return &hosts,nil
}
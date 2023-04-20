package main

import (
	"errors"
	"io/ioutil"
	"strings"

	"gopkg.in/yaml.v2"
)

var config struct {
	BaseDomain   string `yaml:"baseDomain"`
	WikiPath     string `yaml:"wikiPath"`
	StreamPath   string `yaml:"streamPath"`
	DownloadPath string `yaml:"downloadPath"`
	Footer       string `yaml:"footer"`
}

func parseConfig(path string) (err error) {
	if path == "" {
		return errors.New("config path missing. don't forget to provide the environment variable")
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	err = yaml.UnmarshalStrict(data, &config)
	if err != nil {
		return err
	}

	if config.BaseDomain == "" {
		return errors.New("baseDomain config field missing")
	}

	if config.StreamPath == "" {
		return errors.New("streamPath config field missing")
	}

	if config.Footer == "" {
		return errors.New("footer config field missing")
	}

	if config.WikiPath == "" {
		config.WikiPath = config.BaseDomain + "/wiki"
	}

	if config.DownloadPath == "" {
		config.DownloadPath = config.BaseDomain + "/downloads"
	}

	informationTemplate = strings.Replace(informationTemplate, "$$wikiPath$$", config.WikiPath, -1)
	informationTemplate = strings.Replace(informationTemplate, "$$footer$$", config.Footer, -1)

	wikiTemplate = strings.Replace(wikiTemplate, "$$streamPath$$", config.StreamPath, -1)
	wikiTemplate = strings.Replace(wikiTemplate, "$$downloadPath$$", config.DownloadPath, -1)

	return
}

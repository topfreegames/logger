package storage

import (
	"github.com/kelseyhightower/envconfig"
)

const (
	appName = "logger"
)

type esconfig struct {
	Host          string `envconfig:"DEIS_LOGGER_ELASTICSEARCH_SERVICE_HOST" default:"localhost"`
	Port          int    `envconfig:"DEIS_LOGGER_ELASTICSEARCH_SERVICE_PORT" default:"9200"`
	IndexTemplate int    `envconfig:"DEIS_LOGGER_ELASTICSEARCH_INDEX_TEMPLATE" default:"deis-%s"`
}

func parseESConfig(appName string) (*esconfig, error) {
	ret := new(esconfig)
	if err := envconfig.Process(appName, ret); err != nil {
		return nil, err
	}
	return ret, nil
}

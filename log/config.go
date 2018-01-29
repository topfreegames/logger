package log

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

const (
	appName = "logger"
)

// Config defines the log config
type Config struct {
	KafkaBrokers       string `envconfig:"DEIS_KAFKA_BROKERS" default:""`
	KafkaTopic         string `envconfig:"DEIS_KAFKA_TOPIC" default:"log-*"`
	KafkaGroupID       string `envconfig:"DEIS_KAFKA_GROUP_ID" default:"deis-logs-consumer"`
	MessageType        string `envconfig:"DEIS_MESSAGE_TYPE" default:"json"`
	NSQHost            string `envconfig:"DEIS_NSQD_SERVICE_HOST" default:""`
	NSQPort            int    `envconfig:"DEIS_NSQD_SERVICE_PORT_TRANSPORT" default:"4150"`
	NSQTopic           string `envconfig:"NSQ_TOPIC" default:"logs"`
	NSQChannel         string `envconfig:"NSQ_CHANNEL" default:"consume"`
	NSQHandlerCount    int    `envconfig:"NSQ_HANDLER_COUNT" default:"30"`
	StopTimeoutSeconds int    `envconfig:"AGGREGATOR_STOP_TIMEOUT_SEC" default:"1"`
}

func (c Config) nsqURL() string {
	return fmt.Sprintf("%s:%d", c.NSQHost, c.NSQPort)
}

func (c Config) stopTimeoutDuration() time.Duration {
	return time.Duration(c.StopTimeoutSeconds) * time.Second
}

// ParseConfig parses the config
func ParseConfig(appName string) (*Config, error) {
	ret := new(Config)
	if err := envconfig.Process(appName, ret); err != nil {
		return nil, err
	}
	return ret, nil
}

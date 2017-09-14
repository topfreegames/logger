package log

import (
	l "log"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/deis/logger/storage"
)

type kafkaAggregator struct {
	listening      bool
	cfg            *config
	consumer       *kafka.Consumer
	run            bool
	errorChannel   chan error
	storadeAdapter storage.Adapter
}

func newKafkaAggregator(storageAdapter storage.Adapter) Aggregator {
	return &kafkaAggregator{
		errorChannel:   make(chan error),
		storadeAdapter: storageAdapter,
	}
}

// Listen starts the aggregator. Invocations of this function are not concurrency safe and multiple
// serialized invocations have no effect.
func (a *kafkaAggregator) Listen() error {
	// Should only ever be called once
	if !a.listening {
		a.listening = true
		a.run = true

		var err error
		a.cfg, err = parseConfig(appName)
		if err != nil {
			l.Fatalf("config error: %s: ", err)
		}

		c, err := kafka.NewConsumer(&kafka.ConfigMap{
			"bootstrap.servers":               a.cfg.KafkaBrokers,
			"group.id":                        a.cfg.KafkaGroupID,
			"go.events.channel.enable":        true,
			"go.application.rebalance.enable": true,
			"enable.auto.commit":              true,
			"default.topic.config": kafka.ConfigMap{
				"auto.offset.reset":  "latest",
				"auto.commit.enable": true,
			},
		})

		if err != nil {
			l.Fatalf("kafka error: %s: ", err)
		}

		a.consumer = c
		a.consumer.Subscribe(a.cfg.KafkaTopic, nil)

		go func() {
			for a.run == true {
				select {
				case ev := <-a.consumer.Events():
					switch e := ev.(type) {
					case kafka.AssignedPartitions:
						err := a.consumer.Assign(e.Partitions)
						if err != nil {
							l.Println("Failed to assign partitions.")
						}
					case kafka.RevokedPartitions:
						err := a.consumer.Unassign()
						if err != nil {
							l.Println("Failed to unassign partitions.")
						}
					case *kafka.Message:
						handle(e.Value, a.storadeAdapter)
					case kafka.PartitionEOF:
						l.Printf("%% Reached %v\n", e)
					case kafka.Error:
						a.run = false
						a.errorChannel <- e
					default:
						l.Println("ignored kafka event")
					}
				}
			}
		}()
	}
	return nil
}

// Stop is the Aggregator interface implementation
func (a *kafkaAggregator) Stop() error {
	err := a.consumer.Close()
	if err != nil {
		return err
	}
	return nil
}

// Stopped is the Aggregator interface implementation
func (a *kafkaAggregator) Stopped() <-chan error {
	retCh := make(chan error)
	go func() {
		res := <-a.errorChannel
		retCh <- res
	}()
	return retCh
}

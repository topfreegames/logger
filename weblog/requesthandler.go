package weblog

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"

	logger "github.com/deis/logger/log"
	"github.com/deis/logger/storage"
)

const (
	appName = "logger"
)

type requestHandler struct {
	storageAdapter storage.Adapter
}

func newRequestHandler(storageAdapter storage.Adapter) *requestHandler {
	return &requestHandler{
		storageAdapter: storageAdapter,
	}
}

func (h requestHandler) getHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h requestHandler) getLogs(w http.ResponseWriter, r *http.Request) {
	app := mux.Vars(r)["app"]
	var logLines int
	logLinesStr := r.URL.Query().Get("log_lines")
	if logLinesStr == "" {
		log.Printf("The number of lines to return was not specified. Defaulting to 100 lines.")
		logLines = 100
	} else {
		var err error
		logLines, err = strconv.Atoi(logLinesStr)
		if err != nil {
			log.Printf("The specified number of log lines was invalid. Defaulting to 100 lines.")
			logLines = 100
		}
	}
	logs, err := h.storageAdapter.Read(app, logLines)
	if err != nil {
		log.Println(err)
		if strings.HasPrefix(err.Error(), "Could not find logs for") {
			w.WriteHeader(http.StatusNoContent)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
	log.Printf("Returning the last %v lines for %s", logLines, app)
	for _, line := range logs {
		// strip any trailing newline characters from the logs
		fmt.Fprintf(w, "%s\n", strings.TrimSuffix(line, "\n"))
	}
}

func (h requestHandler) deleteLogs(w http.ResponseWriter, r *http.Request) {
	app := mux.Vars(r)["app"]
	if err := h.storageAdapter.Destroy(app); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (h requestHandler) tailLogs(w http.ResponseWriter, r *http.Request) {
	connectionOpen := true
	notify := w.(http.CloseNotifier).CloseNotify()
	read, write := io.Pipe()

	go func() {
		<-notify
		log.Println("Tail connection closed.")
		connectionOpen = false
	}()

	app := mux.Vars(r)["app"]
	process := mux.Vars(r)["process"]

	if process == "" {
		process = ".*"
	}

	cfg, err := logger.ParseConfig(appName)
	if err != nil {
		log.Printf("config error: %s: ", err)
		return
	}

	log.Println("Tail started.")

	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":               cfg.KafkaBrokers,
		"group.id":                        "logs-tail-" + uuid.NewV4().String(),
		"go.events.channel.enable":        true,
		"go.application.rebalance.enable": true,
		"enable.auto.commit":              false,
		"default.topic.config": kafka.ConfigMap{
			"auto.offset.reset":  "latest",
			"auto.commit.enable": false,
		},
	})

	if err != nil {
		log.Printf("kafka error: %s: ", err)
		return
	}

	c.Subscribe("^logs-"+app+"-"+process, nil)

	go func() {
		defer write.Close()
		for connectionOpen == true {
			select {
			case ev := <-c.Events():
				switch e := ev.(type) {
				case kafka.AssignedPartitions:
					err = c.Assign(e.Partitions)
					if err != nil {
						log.Println("[Tail] Failed to assign partitions.")
					}
				case kafka.RevokedPartitions:
					err = c.Unassign()
					if err != nil {
						log.Println("[Tail] Failed to unassign partitions.")
					}
				case *kafka.Message:
					var label string
					var message string

					if cfg.MessageType == "json" {
						label, message = logger.HandleJsonTail(e.Value)
					} else {
						label, message = logger.HandleMsgPackTail(e.Value)
					}

					if label == app {
						fmt.Fprintf(write, "%s\n", strings.TrimSuffix(message, "\n"))
					}
				case kafka.PartitionEOF:
				case kafka.Error:
					connectionOpen = false
				default:
					log.Println("[Tail] ignored kafka event")
				}
			}
		}
	}()
	io.Copy(w, read)

	err = c.Unsubscribe()
	if err != nil {
		log.Printf("kafka unsubscribe error: %s: ", err)
		return
	}

	err = c.Close()
	if err != nil {
		log.Printf("kafka close error: %s: ", err)
		return
	}
	log.Println("Tail Closed.")
}

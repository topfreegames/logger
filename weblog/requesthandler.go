package weblog

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	logger "github.com/deis/logger/log"
	"github.com/deis/logger/storage"
	"github.com/gorilla/mux"
	"github.com/topfreegames/stern/stern"
	"k8s.io/apimachinery/pkg/labels"
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
	process := r.URL.Query().Get("process")
	logs, err := h.storageAdapter.Read(app, logLines, process)
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

var sternCfg *stern.Config

func initStern() {
	cfg, err := logger.ParseConfig(appName)
	if err != nil {
		log.Fatalf("config error: %s: ", err)
	}

	sternCfg = &stern.Config{
		KubeConfig:     cfg.KubeConfigPath,
		ContextName:    cfg.KubeContextName,
		Namespace:      "",
		Writer:         nil,
		PodQuery:       regexp.MustCompile(".*"),
		ContainerQuery: regexp.MustCompile(".*"),
		LabelSelector:  labels.Everything(),
	}
}

func (h requestHandler) tailLogs(w http.ResponseWriter, r *http.Request) {
	reqContext := r.Context()
	app := mux.Vars(r)["app"]
	process := r.URL.Query().Get("process")

	ctx, cancel := context.WithCancel(reqContext)
	go func() {
		<-reqContext.Done()
		log.Println("Tail connection closed.")
		cancel()
	}()

	log.Println("Tail started.")
	go func() {
		cfg := &stern.Config{}
		*cfg = *sternCfg
		cfg.Namespace = app
		cfg.Writer = w
		cfg.WriterMutex = &sync.Mutex{}

		if process != "" {
			cfg.PodQuery = regexp.MustCompile(fmt.Sprintf("-%s-", process))
		}

		err := stern.Run(ctx, cfg)
		if err != nil {
			fmt.Printf("error: %#v\n", err)
		}
	}()
	<-ctx.Done()
	time.Sleep(1000 * time.Millisecond)
	log.Println("Tail Closed.")
}

package weblog

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/deis/logger/storage"
	"github.com/ghostec/stern/stern"
	"github.com/gorilla/mux"
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

func (h requestHandler) tailLogs(w http.ResponseWriter, r *http.Request) {
	notify := w.(http.CloseNotifier).CloseNotify()
	app := mux.Vars(r)["app"]
	// process := r.URL.Query().Get("process")
	reader, writer := io.Pipe()

	ctx, cancel := context.WithCancel(context.Background())
	cfg := &stern.Config{
		KubeConfig:     "/opt/logger/sbin/kubeconfig",
		ContextName:    "kube-stag.tfgco.com",
		Namespace:      app,
		Writer:         writer,
		PodQuery:       regexp.MustCompile(".*"),
		ContainerQuery: regexp.MustCompile(".*"),
	}

	go func() {
		<-notify
		log.Println("Tail connection closed.")
		cancel()
	}()

	// fmt.Fprintf(write, "%s\n", strings.TrimSuffix(message, "\n"))

	log.Println("Tail started.")
	go func() {
		err := stern.Run(ctx, cfg)
		fmt.Printf("error: %#v\n", err)
	}()
	io.Copy(w, reader)
	<-ctx.Done()
	log.Println("Tail Closed.")
}

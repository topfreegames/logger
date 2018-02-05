package weblog

import (
	"github.com/gorilla/mux"
)

func newRouter(rh *requestHandler) *mux.Router {
	r := mux.NewRouter()
	initStern() // tailLogs
	r.HandleFunc("/healthz", rh.getHealthz).Methods("GET")
	r.HandleFunc("/healthz/", rh.getHealthz).Methods("GET")
	r.HandleFunc("/logs/{app}", rh.getLogs).Methods("GET")
	r.HandleFunc("/logs/{app}/", rh.getLogs).Methods("GET")
	r.HandleFunc("/logs/{app}/tail", rh.tailLogs).Methods("GET")
	r.HandleFunc("/logs/{app}/tail/", rh.tailLogs).Methods("GET")
	r.HandleFunc("/logs/{app}", rh.deleteLogs).Methods("DELETE")
	r.HandleFunc("/logs/{app}/", rh.deleteLogs).Methods("DELETE")
	return r
}

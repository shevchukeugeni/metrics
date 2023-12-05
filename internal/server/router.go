package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/shevchukeugeni/metrics/internal/utils"
)

type router struct {
	ms MetricStorage
}

type MetricStorage interface {
	GetMetrics() map[string]utils.Metric
	UpdateMetric(mtype, name, value string) error
}

func SetupRouter(ms MetricStorage) http.Handler {
	ro := &router{
		ms: ms,
	}
	return ro.Handler()
}

func (ro *router) Handler() http.Handler {
	r := http.NewServeMux()
	r.HandleFunc("/metrics", ro.getMetrics)
	r.HandleFunc("/update/", ro.updateMetric)
	return r
}

func (ro *router) getMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "incorrect method", http.StatusMethodNotAllowed)
		return
	}

	metrics := ro.ms.GetMetrics()

	data, err := json.Marshal(metrics)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func (ro *router) updateMetric(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "incorrect method", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/update/")
	if len(path) == 0 || path == "counter/" || path == "gauge/" {
		http.Error(w, "incorrect metric name", http.StatusNotFound)
		return
	}

	params := strings.Split(path, "/")
	if len(params) != 3 {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	err := ro.ms.UpdateMetric(params[0], params[1], params[2])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

package server

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/shevchukeugeni/metrics/internal/store"
	"github.com/shevchukeugeni/metrics/internal/types"
)

const tpl = `
<!DOCTYPE html>
<html lang="en">
<body>
<table>
    <tr>
        <th>Type</th>
        <th>Name</th>
        <th>Value</th>
    </tr>
    {{ range .Metrics}}
        <tr>
            <td>{{ .Metrictype }}</td>
            <td>{{ .Metricname }}</td>
            <td>{{ .Value }}</td>
        </tr>
    {{ end}}
</table>
</body>
</html>`

type router struct {
	logger *zap.Logger
	ms     MetricStorage
}

type MetricStorage interface {
	GetMetrics() map[string]store.Metric
	GetMetric(string) map[string]string
	UpdateMetric(mtype, name, value string) error
}

func SetupRouter(logger *zap.Logger, ms MetricStorage) http.Handler {
	ro := &router{
		logger: logger,
		ms:     ms,
	}
	return ro.Handler()
}

func (ro *router) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(ro.WithLogging)
	r.Get("/", ro.getMetrics)
	r.Get("/value/{mType}/{name}", ro.getMetric)
	r.Post("/update/{mType}/{name}/{value}", ro.updateMetric)
	return r
}

func (ro *router) getMetrics(w http.ResponseWriter, r *http.Request) {
	type metric struct {
		Metrictype string
		Metricname string
		Value      string
	}

	type mdata struct {
		Metrics []metric
	}

	data := mdata{}

	for k, v := range ro.ms.GetMetric("counter") {
		data.Metrics = append(data.Metrics, metric{
			"Counter",
			k,
			v,
		})
	}

	for k, v := range ro.ms.GetMetric("gauge") {
		data.Metrics = append(data.Metrics, metric{
			"Gauge",
			k,
			v,
		})
	}

	tmpl, err := template.New("webpage").Parse(tpl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (ro *router) getMetric(w http.ResponseWriter, r *http.Request) {
	mType := strings.ToLower(chi.URLParam(r, "mType"))
	if mType != types.Counter && mType != types.Gauge {
		http.Error(w, "incorrect metric type", http.StatusNotFound)
		return
	}

	name := chi.URLParam(r, "name")

	metrics := ro.ms.GetMetric(mType)
	if metrics == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	value, ok := metrics[name]
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprint(w, value)
}

func (ro *router) updateMetric(w http.ResponseWriter, r *http.Request) {
	mType := strings.ToLower(chi.URLParam(r, "mType"))
	if mType != "counter" && mType != "gauge" {
		http.Error(w, "incorrect metric type", http.StatusBadRequest)
		return
	}

	name, value := chi.URLParam(r, "name"), chi.URLParam(r, "value")

	err := ro.ms.UpdateMetric(mType, name, value)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (ro *router) WithLogging(h http.Handler) http.Handler {
	logFn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		responseData := &responseData{
			status: 0,
			size:   0,
		}
		lw := LoggingResponseWriter{
			ResponseWriter: w,
			responseData:   responseData,
		}
		h.ServeHTTP(&lw, r)

		duration := time.Since(start)

		ro.logger.Sugar().Infoln(
			"uri", r.RequestURI,
			"method", r.Method,
			"duration", duration,
			"status", responseData.status,
			"size", responseData.size,
		)

	}

	return http.HandlerFunc(logFn)
}

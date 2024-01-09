package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
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
	UpdateMetric(mtype, name, value string) (any, error)
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
	r.Use(gzipMiddleware)
	r.Get("/", ro.getMetrics)
	r.Post("/value/", ro.getMetricJSON)
	r.Post("/update/", ro.updateMetricJSON)
	//DEPRECATED
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

	for k, v := range ro.ms.GetMetric(types.Counter) {
		data.Metrics = append(data.Metrics, metric{
			"Counter",
			k,
			v,
		})
	}

	for k, v := range ro.ms.GetMetric(types.Gauge) {
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
	w.WriteHeader(http.StatusOK)
	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (ro *router) getMetricJSON(w http.ResponseWriter, r *http.Request) {
	var req types.Metrics

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Unable to decode json: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.MType != types.Counter && req.MType != types.Gauge {
		http.Error(w, "incorrect metric type", http.StatusNotFound)
		return
	}

	metrics := ro.ms.GetMetric(req.MType)
	if metrics == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	res := req

	value, ok := metrics[req.ID]
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if req.MType == types.Counter {
		intValue, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			http.Error(w, "Can't parse data: "+err.Error(), http.StatusInternalServerError)
			return
		}
		res.Delta = &intValue
	} else {
		floatValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			http.Error(w, "Can't parse data: "+err.Error(), http.StatusInternalServerError)
			return
		}
		res.Value = &floatValue
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		http.Error(w, "Can't marshal data: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func (ro *router) updateMetricJSON(w http.ResponseWriter, r *http.Request) {
	var req types.Metrics

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Unable to decode json: "+err.Error(), http.StatusBadRequest)
		return
	}

	switch req.MType {
	case types.Counter:
		if req.Delta == nil {
			http.Error(w, "incorrect metric value", http.StatusBadRequest)
			return
		}

		newValue, err := ro.ms.UpdateMetric(req.MType, req.ID, fmt.Sprint(*req.Delta))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		delta := newValue.(int64)
		req.Delta = &delta
	case types.Gauge:
		if req.Value == nil {
			http.Error(w, "incorrect metric value", http.StatusBadRequest)
			return
		}

		newValue, err := ro.ms.UpdateMetric(req.MType, req.ID, fmt.Sprint(*req.Value))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		value := newValue.(float64)
		req.Value = &value
	default:
		http.Error(w, "incorrect metric type", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(req)
	if err != nil {
		http.Error(w, "Can't marshal data: "+err.Error(), http.StatusInternalServerError)
		return
	}
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

func gzipMiddleware(h http.Handler) http.Handler {
	gzipFunc := func(w http.ResponseWriter, r *http.Request) {
		// по умолчанию устанавливаем оригинальный http.ResponseWriter как тот,
		// который будем передавать следующей функции
		ow := w

		// проверяем, что клиент умеет получать от сервера сжатые данные в формате gzip
		acceptEncoding := r.Header.Get("Accept-Encoding")
		supportsGzip := strings.Contains(acceptEncoding, "gzip")
		if supportsGzip {
			// оборачиваем оригинальный http.ResponseWriter новым с поддержкой сжатия
			cw := NewCompressWriter(w)
			// меняем оригинальный http.ResponseWriter на новый
			ow = cw
			// не забываем отправить клиенту все сжатые данные после завершения middleware
			defer cw.Close()
		}

		// проверяем, что клиент отправил серверу сжатые данные в формате gzip
		contentEncoding := r.Header.Get("Content-Encoding")
		sendsGzip := strings.Contains(contentEncoding, "gzip")
		if sendsGzip {
			// оборачиваем тело запроса в io.Reader с поддержкой декомпрессии
			cr, err := NewCompressReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			// меняем тело запроса на новое
			r.Body = cr
			defer cr.Close()
		}

		// передаём управление хендлеру
		h.ServeHTTP(ow, r)
	}

	return http.HandlerFunc(gzipFunc)
}

// DEPRECATED
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

// DEPRECATED
func (ro *router) updateMetric(w http.ResponseWriter, r *http.Request) {
	mType := strings.ToLower(chi.URLParam(r, "mType"))
	if mType != "counter" && mType != "gauge" {
		http.Error(w, "incorrect metric type", http.StatusBadRequest)
		return
	}

	name, value := chi.URLParam(r, "name"), chi.URLParam(r, "value")

	_, err := ro.ms.UpdateMetric(mType, name, value)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

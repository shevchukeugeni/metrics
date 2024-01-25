package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgerrcode"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/avast/retry-go"
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
	dw     *store.DumpWorker
	db     *sql.DB
}

type MetricStorage interface {
	GetMetrics() map[string]store.Metric
	GetMetric(string) map[string]string
	UpdateMetric(mtype, name, value string) (any, error)
	UpdateMetrics([]types.Metrics) error
}

func SetupRouter(logger *zap.Logger, ms MetricStorage, dw *store.DumpWorker, db *sql.DB) http.Handler {
	ro := &router{
		logger: logger,
		ms:     ms,
		dw:     dw,
		db:     db,
	}
	return ro.Handler()
}

func (ro *router) WithRetry(fn func() error, warn string) error {
	interval := time.Second
	return retry.Do(fn,
		retry.Attempts(3),
		retry.Delay(interval),
		retry.OnRetry(func(n uint, err error) {
			ro.logger.Warn(warn, zap.Uint("attempt", n), zap.Error(err))
			interval += 2 * time.Second
		}))
}

func (ro *router) Handler() http.Handler {
	rtr := chi.NewRouter()
	rtr.Use(ro.WithLogging)
	rtr.Get("/ping", ro.dbPing)
	rtr.Group(func(r chi.Router) {
		r.Use(gzipMiddleware)
		r.Get("/", ro.getMetrics)
		r.Post("/value/", ro.getMetricJSON)
		r.Post("/update/", ro.updateMetricJSON)
		r.Post("/updates/", ro.updateMetricsJSON)
	})
	//DEPRECATED
	rtr.Get("/value/{mType}/{name}", ro.getMetric)
	rtr.Post("/update/{mType}/{name}/{value}", ro.updateMetric)
	return rtr
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
			http.Error(w, "Can't parse data: "+err.Error(), http.StatusBadRequest)
			return
		}
		res.Delta = &intValue
	} else {
		floatValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			http.Error(w, "Can't parse data: "+err.Error(), http.StatusBadRequest)
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

	var (
		newValue any
		innerErr error
	)

	switch req.MType {
	case types.Counter:
		if req.Delta == nil {
			http.Error(w, "incorrect metric value", http.StatusBadRequest)
			return
		}

		err = ro.WithRetry(func() error {
			newValue, innerErr = ro.ms.UpdateMetric(req.MType, req.ID, fmt.Sprint(*req.Delta))
			if innerErr != nil {
				if innerErr.Error() == pgerrcode.UniqueViolation {
					return innerErr
				} else {
					return nil
				}
			}
			return nil
		}, "failed to update metric")
		if err != nil {
			http.Error(w, "incorrect metric value", http.StatusBadRequest)
			return
		}
		if innerErr != nil {
			http.Error(w, innerErr.Error(), http.StatusBadRequest)
			return
		}

		delta := newValue.(int64)
		req.Delta = &delta
	case types.Gauge:
		if req.Value == nil {
			http.Error(w, "incorrect metric value", http.StatusBadRequest)
			return
		}

		err = ro.WithRetry(func() error {
			newValue, innerErr = ro.ms.UpdateMetric(req.MType, req.ID, fmt.Sprint(*req.Value))
			if innerErr != nil {
				if innerErr.Error() == pgerrcode.UniqueViolation {
					return innerErr
				} else {
					return nil
				}
			}
			return nil
		}, "failed to update metric")
		if err != nil {
			http.Error(w, "incorrect metric value", http.StatusBadRequest)
			return
		}
		if innerErr != nil {
			http.Error(w, innerErr.Error(), http.StatusBadRequest)
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

	//If DumpWorker was initialized and run in sync mode
	if ro.dw != nil {
		ro.dw.DumpSync()
	}
}

func (ro *router) updateMetricsJSON(w http.ResponseWriter, r *http.Request) {
	var req []types.Metrics

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Unable to decode json: "+err.Error(), http.StatusBadRequest)
		return
	}

	var innerErr error

	err = ro.WithRetry(func() error {
		innerErr = ro.ms.UpdateMetrics(req)
		if innerErr != nil {
			if innerErr.Error() == pgerrcode.UniqueViolation {
				return innerErr
			} else {
				return nil
			}
		}
		return nil
	}, "failed to update metrics")
	if err != nil {
		http.Error(w, "Unable to update batch: "+err.Error(), http.StatusBadRequest)
		return
	}
	if innerErr != nil {
		http.Error(w, "Unable to update batch: "+innerErr.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	//If DumpWorker was initialized and run in sync mode
	if ro.dw != nil {
		ro.dw.DumpSync()
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
			//	оборачиваем оригинальный http.ResponseWriter новым с поддержкой сжатия
			cw := NewCompressWriter(w)
			//меняем оригинальный http.ResponseWriter на новый
			ow = cw
			//не забываем отправить клиенту все сжатые данные после завершения middleware
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

func (ro *router) dbPing(w http.ResponseWriter, r *http.Request) {
	err := ro.db.Ping()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

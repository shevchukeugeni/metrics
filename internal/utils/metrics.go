package utils

import (
	"fmt"
	"math/rand"
	"reflect"
	"runtime"
)

type RuntimeMetrics struct {
	Gauge   map[string]string
	Counter map[string]int64
}

func NewRuntimeMetrics() *RuntimeMetrics {
	return &RuntimeMetrics{
		Gauge:   make(map[string]string),
		Counter: make(map[string]int64),
	}
}

func (rm *RuntimeMetrics) Update() {
	memStats := new(runtime.MemStats)

	runtime.ReadMemStats(memStats)

	values := reflect.ValueOf(*memStats)
	typesOfV := values.Type()

	for i := 0; i < values.NumField(); i++ {
		switch typesOfV.Field(i).Name {
		case "PauseNs", "PauseEnd", "BySize", "EnableGC", "DebugGC":
			continue
		default:
			rm.Gauge[typesOfV.Field(i).Name] = fmt.Sprint(values.Field(i).Interface())
		}
	}

	rm.Gauge["RandomValue"] = fmt.Sprint(rand.Float64())

	rm.Counter["PollCount"]++
}

package store

import (
	"fmt"
	"math/rand"
	"reflect"
	"runtime"
	"strconv"
)

type RuntimeMetrics struct {
	Gauge   map[string]float64
	Counter map[string]int64
}

func NewRuntimeMetrics() *RuntimeMetrics {
	return &RuntimeMetrics{
		Gauge:   make(map[string]float64),
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
			val, err := strconv.ParseFloat(fmt.Sprint(values.Field(i).Interface()), 64)
			if err != nil {
				continue
			}
			rm.Gauge[typesOfV.Field(i).Name] = val
		}
	}

	rm.Gauge["RandomValue"] = rand.Float64()

	rm.Counter["PollCount"]++
}

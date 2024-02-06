package store

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"reflect"
	"runtime"
	"strconv"
	"sync"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

type Metrics struct {
	Gauge   map[string]float64
	Counter map[string]int64
	mu      sync.Mutex
}

func NewMetrics() *Metrics {
	return &Metrics{
		Gauge:   make(map[string]float64),
		Counter: make(map[string]int64),
	}
}

func (rm *Metrics) UpdateRuntime(wg *sync.WaitGroup) {
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
			rm.mu.Lock()
			rm.Gauge[typesOfV.Field(i).Name] = val
			rm.mu.Unlock()
		}
	}

	rm.mu.Lock()
	rm.Gauge["RandomValue"] = rand.Float64()

	rm.Counter["PollCount"]++
	rm.mu.Unlock()

	wg.Done()
}

func (rm *Metrics) UpdateMemory(wg *sync.WaitGroup) {
	vm, err := mem.VirtualMemoryWithContext(context.Background())
	if err != nil {
		log.Printf("Unable to get virtual memory stat: %s", err.Error())
	}

	cpuInfo, err := cpu.Counts(true)
	if err != nil {
		log.Printf("Unable to get cpu stat: %s", err.Error())
	}

	rm.mu.Lock()
	rm.Gauge["TotalMemory"] = float64(vm.Total)
	rm.Gauge["FreeMemory"] = float64(vm.Free)
	rm.Gauge["CPUutilization1"] = float64(cpuInfo)
	rm.mu.Unlock()

	wg.Done()
}

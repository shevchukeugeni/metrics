package store

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/shevchukeugeni/metrics/internal/types"
)

type MemStorage struct {
	metrics map[string]Metric
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		map[string]Metric{
			types.Gauge:   Gauge{},
			types.Counter: Counter{},
		},
	}
}

func (ms *MemStorage) GetMetric(mtype string) map[string]string {
	metric, ok := ms.metrics[mtype]
	if !ok {
		return nil
	} else {
		return metric.Get()
	}
}

func (ms *MemStorage) GetMetrics() map[string]Metric {
	return ms.metrics
}

func (ms *MemStorage) UpdateMetric(mtype, name, value string) (any, error) {
	metric, ok := ms.metrics[mtype]
	if !ok {
		return nil, types.ErrUnknownType
	}

	return metric.Update(name, value)
}

type Metric interface {
	Get() map[string]string
	Update(name, value string) (any, error)
}

type Gauge map[string]float64

func (g Gauge) Get() map[string]string {
	strMap := make(map[string]string)
	for k, v := range g {
		strMap[k] = fmt.Sprint(v)
	}

	return strMap
}

func (g Gauge) Update(name, value string) (any, error) {
	fValue, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil, err
	}

	if name == "" {
		return nil, errors.New("incorrect name")
	}

	g[name] = fValue
	return fValue, nil
}

type Counter map[string]int64

func (c Counter) Get() map[string]string {
	strMap := make(map[string]string)
	for k, v := range c {
		strMap[k] = fmt.Sprint(v)
	}

	return strMap
}

func (c Counter) Update(name, value string) (any, error) {
	iValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return nil, err
	}

	if name == "" {
		return nil, errors.New("incorrect name")
	}

	c[name] += iValue
	return c[name], nil
}

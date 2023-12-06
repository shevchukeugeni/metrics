package utils

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/shevchukeugeni/metrics/internal/types"
)

type MemStorage struct {
	Metrics map[string]Metric
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		map[string]Metric{
			"gauge":   Gauge{},
			"counter": Counter{},
		},
	}
}

func (ms *MemStorage) GetMetric(mtype string) map[string]string {
	metric, ok := ms.Metrics[mtype]
	if !ok {
		return nil
	} else {
		return metric.Get()
	}
}

func (ms *MemStorage) GetMetrics() map[string]Metric {
	return ms.Metrics
}

func (ms *MemStorage) UpdateMetric(mtype, name, value string) error {
	metric, ok := ms.Metrics[mtype]
	if !ok {
		return types.ErrUnknownType
	}

	return metric.Update(name, value)
}

type Metric interface {
	Get() map[string]string
	Update(name, value string) error
}

type Gauge map[string]float64

func (g Gauge) Get() map[string]string {
	strMap := make(map[string]string)
	for k, v := range g {
		strMap[k] = fmt.Sprint(v)
	}

	return strMap
}

func (g Gauge) Update(name, value string) error {
	fValue, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return err
	}

	if name == "" {
		return errors.New("incorrect name")
	}

	g[name] = fValue
	return nil
}

type Counter map[string]int64

func (c Counter) Get() map[string]string {
	strMap := make(map[string]string)
	for k, v := range c {
		strMap[k] = fmt.Sprint(v)
	}

	return strMap
}

func (c Counter) Update(name, value string) error {
	iValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return err
	}

	if name == "" {
		return errors.New("incorrect name")
	}

	c[name] += iValue
	return nil
}

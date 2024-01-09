package store

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"time"

	"go.uber.org/zap"

	"github.com/shevchukeugeni/metrics/internal/types"
)

type DumpWorker struct {
	logger   *zap.Logger
	cfg      *types.DumpConfig
	storage  *MemStorage
	syncMode bool
}

type dumpData struct {
	Metrics []metric
}

type metric struct {
	MType string `json:"type"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

func NewDumpWorker(logger *zap.Logger, cfg *types.DumpConfig, storage *MemStorage) *DumpWorker {
	if cfg.FileStoragePath == "" {
		logger.Info("Dumping to file disabled")
		return nil
	}

	if cfg.Restore {
		data, err := restore(cfg.FileStoragePath)
		if err != nil || data == nil {
			logger.Error("failed to restore", zap.Error(err))
		} else {
			for _, m := range data.Metrics {
				_, err = storage.UpdateMetric(m.MType, m.Name, m.Value)
				if err != nil {
					logger.Error("failed to restore", zap.Error(err))
				}
			}
		}
	}

	return &DumpWorker{
		logger:   logger,
		cfg:      cfg,
		storage:  storage,
		syncMode: cfg.StoreInterval == 0,
	}
}

func (dw *DumpWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(dw.cfg.StoreInterval) * time.Second)
	for {
		select {
		case <-ctx.Done():
			dw.dump()
			return
		case <-ticker.C:
			dw.logger.Info("dumping to file")
			dw.dump()
		}
	}
}

func (dw *DumpWorker) DumpSync() {
	if dw.syncMode {
		dw.dump()
	}
}

func (dw *DumpWorker) dump() {
	var dmp dumpData
	counters := dw.storage.GetMetric(types.Counter)
	for k, v := range counters {
		dmp.Metrics = append(dmp.Metrics, metric{MType: types.Counter, Name: k, Value: v})
	}
	gauges := dw.storage.GetMetric(types.Gauge)
	for k, v := range gauges {
		dmp.Metrics = append(dmp.Metrics, metric{MType: types.Gauge, Name: k, Value: v})
	}

	data, err := json.MarshalIndent(dmp, "", "   ")
	if err != nil {
		dw.logger.Error("failed to marshal json", zap.Error(err))
	}

	err = os.WriteFile(dw.cfg.FileStoragePath, data, 0666)
	if err != nil {
		dw.logger.Error("failed to save json", zap.Error(err))
	}
}

func restore(path string) (*dumpData, error) {
	jsonFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer jsonFile.Close()

	jsonData, err := io.ReadAll(jsonFile)
	if err != nil {
		return nil, err
	}
	data := &dumpData{}
	if err := json.Unmarshal(jsonData, data); err != nil {
		return nil, err
	}
	return data, nil
}

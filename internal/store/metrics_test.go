package store

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRuntimeMetrics_Update(t *testing.T) {
	type fields struct {
		Gauge   map[string]float64
		Counter map[string]int64
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "success",
			fields: fields{
				Gauge:   make(map[string]float64),
				Counter: make(map[string]int64),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rm := &Metrics{
				Gauge:   tt.fields.Gauge,
				Counter: tt.fields.Counter,
			}
			wg := new(sync.WaitGroup)

			wg.Add(1)
			go rm.UpdateRuntime(wg)
			wg.Wait()

			require.NotEqual(t, 0, len(rm.Gauge))
			require.Equal(t, int64(1), rm.Counter["PollCount"])
		})
	}
}

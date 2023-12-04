package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRuntimeMetrics_Update(t *testing.T) {
	type fields struct {
		Gauge   map[string]string
		Counter map[string]int64
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "success",
			fields: fields{
				Gauge:   make(map[string]string),
				Counter: make(map[string]int64),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rm := &RuntimeMetrics{
				Gauge:   tt.fields.Gauge,
				Counter: tt.fields.Counter,
			}
			rm.Update()

			require.NotEqual(t, 0, len(rm.Gauge))
			require.Equal(t, int64(1), rm.Counter["PollCount"])
		})
	}
}

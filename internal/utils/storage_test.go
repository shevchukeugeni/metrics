package utils

import (
	"testing"
)

func TestCounter_Update(t *testing.T) {
	type args struct {
		name  string
		value string
	}

	tests := []struct {
		name    string
		g       Counter
		args    args
		wantErr bool
	}{
		{
			name: "correct value",
			g:    map[string]int64{},
			args: args{
				name:  "test1",
				value: "1",
			},
			wantErr: false,
		},
		{
			name: "correct with negative",
			g:    map[string]int64{},
			args: args{
				name:  "test1",
				value: "-11",
			},
			wantErr: false,
		},
		{
			name: "float value",
			g:    map[string]int64{},
			args: args{
				name:  "test1",
				value: "1.5",
			},
			wantErr: true,
		},
		{
			name: "empty name",
			g:    map[string]int64{},
			args: args{
				name:  "",
				value: "1",
			},
			wantErr: true,
		},
		{
			name: "empty value",
			g:    map[string]int64{},
			args: args{
				name:  "test1",
				value: "",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.g.Update(tt.args.name, tt.args.value); (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

}

func TestGauge_Update(t *testing.T) {
	type args struct {
		name  string
		value string
	}
	tests := []struct {
		name    string
		g       Gauge
		args    args
		wantErr bool
	}{
		{
			name: "correct value",
			g:    map[string]float64{},
			args: args{
				name:  "test1",
				value: "1.5",
			},
			wantErr: false,
		},
		{
			name: "correct with negative",
			g:    map[string]float64{},
			args: args{
				name:  "test1",
				value: "-11.5",
			},
			wantErr: false,
		},
		{
			name: "string value",
			g:    map[string]float64{},
			args: args{
				name:  "test1",
				value: "some string",
			},
			wantErr: true,
		},
		{
			name: "empty name",
			g:    map[string]float64{},
			args: args{
				name:  "",
				value: "1",
			},
			wantErr: true,
		},
		{
			name: "empty value",
			g:    map[string]float64{},
			args: args{
				name:  "test1",
				value: "",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.g.Update(tt.args.name, tt.args.value); (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMemStorage_UpdateMetric(t *testing.T) {
	type fields struct {
		Metrics map[string]Metric
	}
	type args struct {
		mtype string
		name  string
		value string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "correct case",
			fields: fields{map[string]Metric{
				"gauge": Gauge{
					"test1": 0.5,
				},
				"counter": Counter{
					"test2": 4,
				},
			},
			},
			args: args{
				mtype: "counter",
				name:  "test2",
				value: "1",
			},
			wantErr: false,
		},
		{
			name: "incorrect metric type",
			fields: fields{map[string]Metric{
				"gauge": Gauge{
					"test1": 0.5,
				},
				"counter": Counter{
					"test2": 4,
				},
			},
			},
			args: args{
				mtype: "counter-f",
				name:  "test2",
				value: "1",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := &MemStorage{
				Metrics: tt.fields.Metrics,
			}
			if err := ms.UpdateMetric(tt.args.mtype, tt.args.name, tt.args.value); (err != nil) != tt.wantErr {
				t.Errorf("UpdateMetric() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

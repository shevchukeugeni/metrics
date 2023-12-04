package server

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shevchukeugeni/metrics/internal/mocks"
	"github.com/shevchukeugeni/metrics/internal/utils"
)

func Test_router_updateMetric(t *testing.T) {
	type want struct {
		code          int
		emptyResponse bool
		response      string
		contentType   string
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockStorage := mocks.NewMockMetricStorage(mockCtrl)

	mockStorage.EXPECT().UpdateMetric("counter", "test", "1").Return(nil).Times(1)
	mockStorage.EXPECT().UpdateMetric("gauge", "test", "1").Return(nil).Times(1)
	mockStorage.EXPECT().UpdateMetric("gauge", "test", "2").Return(errors.New("Bad request")).Times(1)

	tests := []struct {
		name   string
		method string
		target string
		want   want
	}{
		{
			name:   "positive test #1",
			method: http.MethodPost,
			target: "/update/counter/test/1",
			want: want{
				code:          200,
				emptyResponse: true,
				contentType:   "",
			},
		},
		{
			name:   "positive test #2",
			method: http.MethodPost,
			target: "/update/gauge/test/1",
			want: want{
				code:          200,
				emptyResponse: true,
				contentType:   "",
			},
		},
		{
			name:   "failed test #1",
			method: http.MethodPost,
			target: "/update/",
			want: want{
				code:          404,
				emptyResponse: false,
				response:      "incorrect metric name\n",
				contentType:   "text/plain; charset=utf-8",
			},
		},
		{
			name:   "failed test #2",
			method: http.MethodGet,
			target: "/update/counter/1/1",
			want: want{
				code:          405,
				emptyResponse: false,
				response:      "incorrect method\n",
				contentType:   "text/plain; charset=utf-8",
			},
		},
		{
			name:   "failed test #3",
			method: http.MethodPost,
			target: "/update/less/params",
			want: want{
				code:          400,
				emptyResponse: false,
				response:      "Bad request\n",
				contentType:   "text/plain; charset=utf-8",
			},
		},
		{
			name:   "failed test #4",
			method: http.MethodPost,
			target: "/update/gauge/test/2",
			want: want{
				code:          400,
				emptyResponse: false,
				response:      "Bad request\n",
				contentType:   "text/plain; charset=utf-8",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ro := &router{
				ms: mockStorage,
			}

			request := httptest.NewRequest(tt.method, tt.target, nil)
			// создаём новый Recorder
			w := httptest.NewRecorder()

			ro.updateMetric(w, request)

			res := w.Result()

			// проверяем код ответа
			assert.Equal(t, tt.want.code, res.StatusCode)
			// получаем и проверяем тело запроса
			defer res.Body.Close()
			resBody, err := io.ReadAll(res.Body)

			require.NoError(t, err)

			if tt.want.emptyResponse {
				require.Empty(t, resBody)
			} else {
				assert.Equal(t, tt.want.response, string(resBody))
			}

			assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))

		})
	}
}

func Test_router_getMetrics(t *testing.T) {
	type want struct {
		code        int
		response    string
		contentType string
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockStorage := mocks.NewMockMetricStorage(mockCtrl)

	mockStorage.EXPECT().GetMetrics().Return(
		map[string]utils.Metric{
			"gauge": utils.Gauge{
				"test1": 0.5,
			},
			"counter": utils.Counter{
				"test2": 4,
			},
		}).Times(1)
	mockStorage.EXPECT().GetMetrics().Return(nil).Times(1)

	tests := []struct {
		name    string
		method  string
		storage MetricStorage
		want    want
	}{
		{
			name:    "positive test #1",
			method:  http.MethodGet,
			storage: mockStorage,
			want: want{
				code:        200,
				response:    `{"counter":{"test2":4},"gauge":{"test1":0.5}}`,
				contentType: "application/json",
			},
		},
		{
			name:    "incorrect method",
			method:  http.MethodPost,
			storage: mockStorage,
			want: want{
				code:        405,
				response:    "incorrect method\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:   "not initialized",
			method: http.MethodGet,
			want: want{
				code:        200,
				response:    "null",
				contentType: "application/json",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ro := &router{
				ms: mockStorage,
			}

			request := httptest.NewRequest(tt.method, "/metrics", nil)
			// создаём новый Recorder
			w := httptest.NewRecorder()

			ro.getMetrics(w, request)

			res := w.Result()

			// проверяем код ответа
			assert.Equal(t, tt.want.code, res.StatusCode)
			// получаем и проверяем тело запроса
			defer res.Body.Close()
			resBody, err := io.ReadAll(res.Body)

			require.NoError(t, err)

			assert.Equal(t, tt.want.response, string(resBody))

			assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))

		})
	}
}

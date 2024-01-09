package server

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/shevchukeugeni/metrics/internal/mocks"
)

var logger = zap.L()

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

	mockStorage.EXPECT().UpdateMetric("counter", "test", "1").Return(int64(1), nil).Times(1)
	mockStorage.EXPECT().UpdateMetric("gauge", "test", "1").Return(float64(2), nil).Times(1)
	mockStorage.EXPECT().UpdateMetric("gauge", "test", "2").Return(nil, errors.New("Bad request")).Times(1)

	ts := httptest.NewServer(SetupRouter(logger, mockStorage))
	defer ts.Close()

	tests := []struct {
		name   string
		method string
		target string
		body   []byte
		want   want
	}{
		{
			name:   "positive test #1",
			method: http.MethodPost,
			target: "/update/",
			body:   []byte(`{"id":"test","type":"counter","delta":1}`),
			want: want{
				code:          200,
				emptyResponse: false,
				response:      "{\"id\":\"test\",\"type\":\"counter\",\"delta\":1}\n",
				contentType:   "application/json",
			},
		},
		{
			name:   "positive test #2",
			method: http.MethodPost,
			target: "/update/",
			body:   []byte(`{"id":"test","type":"gauge","value":1}`),
			want: want{
				code:          200,
				emptyResponse: false,
				response:      "{\"id\":\"test\",\"type\":\"gauge\",\"value\":2}\n",
				contentType:   "application/json",
			},
		},
		{
			name:   "failed test #1",
			method: http.MethodPost,
			target: "/update/134",
			want: want{
				code:          404,
				emptyResponse: false,
				response:      "404 page not found\n",
				contentType:   "text/plain; charset=utf-8",
			},
		},
		{
			name:   "failed test #2",
			method: http.MethodGet,
			target: "/update/",
			want: want{
				code:          405,
				emptyResponse: false,
				response:      "",
				contentType:   "",
			},
		},
		{
			name:   "failed test #3",
			method: http.MethodPost,
			target: "/update/",
			body:   []byte(`{"id":"test","type":"counter"}`),
			want: want{
				code:          400,
				emptyResponse: false,
				response:      "incorrect metric value\n\x1f\x8b\b\x00\x00\x00\x00\x00\x00\xff\x01\x00\x00\xff\xff\x00\x00\x00\x00\x00\x00\x00\x00",
				contentType:   "text/plain; charset=utf-8",
			},
		},
		{
			name:   "failed test #4",
			method: http.MethodPost,
			target: "/update/",
			body:   []byte(`{"id":"test","type":"gauge"}`),
			want: want{
				code:          400,
				emptyResponse: false,
				response:      "incorrect metric value\n\x1f\x8b\b\x00\x00\x00\x00\x00\x00\xff\x01\x00\x00\xff\xff\x00\x00\x00\x00\x00\x00\x00\x00",
				contentType:   "text/plain; charset=utf-8",
			},
		},
		{
			name:   "failed test #5 incorrect metric",
			method: http.MethodPost,
			target: "/update/",
			body:   []byte(`{"id":"test","type":"gauge","value":2}`),
			want: want{
				code:          400,
				emptyResponse: false,
				response:      "Bad request\n\x1f\x8b\b\x00\x00\x00\x00\x00\x00\xff\x01\x00\x00\xff\xff\x00\x00\x00\x00\x00\x00\x00\x00",
				contentType:   "text/plain; charset=utf-8",
			},
		},
		{
			name:   "failed test #6 incorrect metric type",
			method: http.MethodPost,
			target: "/update/",
			body:   []byte(`{"id":"test","type":"couunter","delta":1}`),
			want: want{
				code:          404,
				emptyResponse: false,
				response:      "incorrect metric type\n\x1f\x8b\b\x00\x00\x00\x00\x00\x00\xff\x01\x00\x00\xff\xff\x00\x00\x00\x00\x00\x00\x00\x00",
				contentType:   "text/plain; charset=utf-8",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, body := testRequest(t, ts, tt.method, tt.target, tt.body)
			defer res.Body.Close()
			assert.Equal(t, tt.want.code, res.StatusCode)

			if tt.want.emptyResponse {
				require.Empty(t, body)
			} else {
				assert.Equal(t, tt.want.response, string(body))
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

	mockStorage.EXPECT().GetMetric("counter").Return(
		map[string]string{
			"test2": "4",
		}).Times(1)
	mockStorage.EXPECT().GetMetric("gauge").Return(nil).Times(1)

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
				response:    "\n<!DOCTYPE html>\n<html lang=\"en\">\n<body>\n<table>\n    <tr>\n        <th>Type</th>\n        <th>Name</th>\n        <th>Value</th>\n    </tr>\n    \n        <tr>\n            <td>Counter</td>\n            <td>test2</td>\n            <td>4</td>\n        </tr>\n    \n</table>\n</body>\n</html>",
				contentType: "text/html; charset=UTF-8",
			},
		},
		{
			name:    "incorrect method",
			method:  http.MethodPost,
			storage: mockStorage,
			want: want{
				code:        405,
				response:    "",
				contentType: "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(SetupRouter(logger, tt.storage))
			defer ts.Close()

			res, body := testRequest(t, ts, tt.method, "/", nil)
			defer res.Body.Close()
			assert.Equal(t, tt.want.code, res.StatusCode)

			assert.Equal(t, tt.want.response, body)

			assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
		})
	}
}

func Test_router_getMetric(t *testing.T) {
	type want struct {
		code        int
		response    string
		contentType string
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockStorage := mocks.NewMockMetricStorage(mockCtrl)

	mockStorage.EXPECT().GetMetric("counter").Return(
		map[string]string{
			"test1": "1",
		}).Times(1)
	mockStorage.EXPECT().GetMetric("gauge").Return(
		map[string]string{
			"test2": "2.22",
		}).Times(1)

	ts := httptest.NewServer(SetupRouter(logger, mockStorage))
	defer ts.Close()

	tests := []struct {
		name   string
		method string
		target string
		body   []byte
		want   want
	}{
		{
			name:   "positive test #1",
			method: http.MethodPost,
			target: "/value/",
			body:   []byte(`{"id":"test1","type":"counter"}`),
			want: want{
				code:        200,
				response:    "{\"id\":\"test1\",\"type\":\"counter\",\"delta\":1}\n",
				contentType: "application/json",
			},
		},
		{
			name:   "positive test #2",
			method: http.MethodPost,
			target: "/value/",
			body:   []byte(`{"id":"test2","type":"gauge"}`),
			want: want{
				code:        200,
				response:    "{\"id\":\"test2\",\"type\":\"gauge\",\"value\":2.22}\n",
				contentType: "application/json",
			},
		},
		{
			name:   "failed test #1",
			method: http.MethodGet,
			target: "/values/",
			want: want{
				code:        404,
				response:    "404 page not found\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:   "failed test #2",
			method: http.MethodGet,
			target: "/value/",
			want: want{
				code:        405,
				response:    "",
				contentType: "",
			},
		},
		{
			name:   "failed test #3",
			method: http.MethodGet,
			target: "/valuee/",
			want: want{
				code:        404,
				response:    "404 page not found\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, body := testRequest(t, ts, tt.method, tt.target, tt.body)
			defer res.Body.Close()

			assert.Equal(t, tt.want.code, res.StatusCode)
			assert.Equal(t, tt.want.response, body)
			assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
		})
	}
}

func testRequest(t *testing.T, ts *httptest.Server,
	method, path string, body []byte) (*http.Response, string) {
	bodyReader := bytes.NewReader(body)

	req, err := http.NewRequest(method, ts.URL+path, bodyReader)
	require.NoError(t, err)

	resp, err := ts.Client().Do(req)
	require.NoError(t, err)

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp, string(respBody)
}

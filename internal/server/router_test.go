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

	mockStorage.EXPECT().UpdateMetric("counter", "test", "1").Return(nil).Times(1)
	mockStorage.EXPECT().UpdateMetric("gauge", "test", "1").Return(nil).Times(1)
	mockStorage.EXPECT().UpdateMetric("gauge", "test", "2").Return(errors.New("Bad request")).Times(1)

	ts := httptest.NewServer(SetupRouter(logger, mockStorage))
	defer ts.Close()

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
				response:      "404 page not found\n",
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
				response:      "",
				contentType:   "",
			},
		},
		{
			name:   "failed test #3",
			method: http.MethodPost,
			target: "/update/less/params",
			want: want{
				code:          404,
				emptyResponse: false,
				response:      "404 page not found\n",
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
		{
			name:   "failed test #5 incorrect metric type",
			method: http.MethodPost,
			target: "/update/gauga/test/2",
			want: want{
				code:          400,
				emptyResponse: false,
				response:      "incorrect metric type\n",
				contentType:   "text/plain; charset=utf-8",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, body := testRequest(t, ts, tt.method, tt.target)
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

			res, body := testRequest(t, ts, tt.method, "/")
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
		}).Times(2)

	ts := httptest.NewServer(SetupRouter(logger, mockStorage))
	defer ts.Close()

	tests := []struct {
		name   string
		method string
		target string
		want   want
	}{
		{
			name:   "positive test #1",
			method: http.MethodGet,
			target: "/value/counter/test1",
			want: want{
				code:        200,
				response:    "1",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:   "positive test #2",
			method: http.MethodGet,
			target: "/value/gauge/test2",
			want: want{
				code:        200,
				response:    "2.22",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:   "failed test #1",
			method: http.MethodGet,
			target: "/value/",
			want: want{
				code:        404,
				response:    "404 page not found\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:   "failed test #2",
			method: http.MethodPost,
			target: "/value/counter/1",
			want: want{
				code:        405,
				response:    "",
				contentType: "",
			},
		},
		{
			name:   "failed test #3",
			method: http.MethodGet,
			target: "/value/gauge/test",
			want: want{
				code:        404,
				response:    "not found\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, body := testRequest(t, ts, tt.method, tt.target)
			defer res.Body.Close()

			assert.Equal(t, tt.want.code, res.StatusCode)
			assert.Equal(t, tt.want.response, body)
			assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
		})
	}
}

func testRequest(t *testing.T, ts *httptest.Server, method,
	path string) (*http.Response, string) {
	req, err := http.NewRequest(method, ts.URL+path, nil)
	require.NoError(t, err)

	resp, err := ts.Client().Do(req)
	require.NoError(t, err)

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp, string(respBody)
}

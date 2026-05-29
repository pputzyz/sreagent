package datasource

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"

	"github.com/sreagent/sreagent/internal/pkg/safehttp"
)

// RemoteWriteClient writes time series data to a Prometheus-compatible remote_write endpoint.
// It supports Prometheus and VictoriaMetrics (which accepts the same protocol).
type RemoteWriteClient struct {
	httpClient *http.Client
	endpoint   string
	authType   string
	authConfig string
}

// NewRemoteWriteClient creates a new client for writing time series via remote_write.
// Uses NewInternalClient to allow private addresses (datasources are internal).
func NewRemoteWriteClient(endpoint, authType, authConfig string, timeout time.Duration) *RemoteWriteClient {
	return &RemoteWriteClient{
		httpClient: safehttp.NewInternalClient(timeout),
		endpoint:   endpoint,
		authType:   authType,
		authConfig: authConfig,
	}
}

// Write sends time series samples via Prometheus remote_write protocol (snappy-compressed protobuf).
func (c *RemoteWriteClient) Write(ctx context.Context, series []prompb.TimeSeries) error {
	if len(series) == 0 {
		return nil
	}

	req := &prompb.WriteRequest{Timeseries: series}
	data, err := req.Marshal()
	if err != nil {
		return fmt.Errorf("marshal write request: %w", err)
	}

	compressed := snappy.Encode(nil, data)

	url := c.endpoint + "/api/v1/write"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(compressed))
	if err != nil {
		return fmt.Errorf("create remote write request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/x-protobuf")
	httpReq.Header.Set("Content-Encoding", "snappy")
	httpReq.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")

	if err := applyAuth(httpReq, c.authType, c.authConfig); err != nil {
		return fmt.Errorf("remote write auth: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("remote write request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode/100 != 2 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("remote write failed: status=%d body=%s", resp.StatusCode, string(body))
	}

	return nil
}

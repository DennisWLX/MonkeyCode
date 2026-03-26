package loki

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type Client struct {
	baseURL string
	client  *http.Client
	logger  *slog.Logger
}

func NewClient(baseURL string, logger *slog.Logger) *Client {
	return &Client{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 10 * time.Second},
		logger:  logger,
	}
}

type LogEntry struct {
	Timestamp int64  `json:"timestamp"`
	Line      string `json:"line"`
}

type Stream struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"`
}

type PushRequest struct {
	Streams []Stream `json:"streams"`
}

func (c *Client) Push(ctx context.Context, taskID string, entries []LogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	stream := Stream{
		Stream: map[string]string{
			"task_id": taskID,
		},
		Values: make([][]string, 0, len(entries)),
	}

	for _, e := range entries {
		stream.Values = append(stream.Values, []string{
			fmt.Sprintf("%d", e.Timestamp),
			e.Line,
		})
	}

	req := PushRequest{
		Streams: []Stream{stream},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal push request: %w", err)
	}

	url := c.baseURL + "/loki/api/v1/push"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("loki push failed: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	return nil
}

package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/chaitin/MonkeyCode/taskflow/internal/config"
)

type Client struct {
	client *http.Client
	addr   string
}

func NewClient(cfg *config.BackendConfig) *Client {
	return &Client{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		addr: cfg.Addr,
	}
}

type Token struct {
	Kind  string     `json:"kind"`
	User  *TokenUser `json:"user,omitempty"`
	Token string     `json:"token"`
}

type TokenUser struct {
	ID        string      `json:"id"`
	Name      string      `json:"name,omitempty"`
	AvatarURL string      `json:"avatar_url,omitempty"`
	Email     string      `json:"email,omitempty"`
	Team      *TokenTeam  `json:"team,omitempty"`
}

type TokenTeam struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

type CheckTokenReq struct {
	Token     string `json:"token"`
	MachineID string `json:"machine_id,omitempty"`
}

type HostInfo struct {
	ID         string `json:"id"`
	UserID     string `json:"user_id"`
	Hostname   string `json:"hostname"`
	Arch       string `json:"arch"`
	OS         string `json:"os"`
	Name       string `json:"name"`
	Cores      int32  `json:"cores"`
	Memory     uint64 `json:"memory"`
	Disk       uint64 `json:"disk"`
	PublicIP   string `json:"public_ip"`
	InternalIP string `json:"internal_ip"`
	Version    string `json:"version"`
	CreatedAt  int64  `json:"created_at"`
}

type Response[T any] struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data T      `json:"data"`
}

func (c *Client) CheckToken(ctx context.Context, req *CheckTokenReq) (*Token, error) {
	resp, err := c.post(ctx, "/internal/check-token", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result Response[*Token]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("check token failed: %s", result.Msg)
	}

	return result.Data, nil
}

func (c *Client) ReportHostInfo(ctx context.Context, host *HostInfo) error {
	resp, err := c.post(ctx, "/internal/host-info", host)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result Response[any]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Code != 0 {
		return fmt.Errorf("report host info failed: %s", result.Msg)
	}

	return nil
}

func (c *Client) post(ctx context.Context, path string, body any) (*http.Response, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s%s", c.addr, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return resp, nil
}

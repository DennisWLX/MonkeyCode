package github

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v74/github"
)

var pemBlockType = "RSA PRIVATE KEY"

// GitHubApp GitHub App 客户端
type GitHubApp struct {
	appID      int64
	privateKey []byte
	logger     *slog.Logger
}

// NewGitHubApp 创建 GitHub App 客户端
func NewGitHubApp(appID int64, privateKey string, logger *slog.Logger) (*GitHubApp, error) {
	formattedKey, err := formatPrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("invalid private key format: %w", err)
	}

	l := logger.With("module", "github_app")
	l.Info("github app client created", "app_id", appID, "key_length", len(formattedKey))

	return &GitHubApp{
		appID:      appID,
		privateKey: formattedKey,
		logger:     l,
	}, nil
}

func formatPrivateKey(key string) ([]byte, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("private key is empty")
	}

	if !strings.HasPrefix(key, "-----BEGIN") {
		return nil, fmt.Errorf("key must be PEM encoded (missing -----BEGIN marker)")
	}

	block, _ := pem.Decode([]byte(key))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	if block.Type != "RSA PRIVATE KEY" && block.Type != "PRIVATE KEY" {
		return nil, fmt.Errorf("unexpected PEM block type: %s (expected RSA PRIVATE KEY or PRIVATE KEY)", block.Type)
	}

	var parsedKey interface{}
	var err error

	if block.Type == "PRIVATE KEY" {
		parsedKey, err = x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse PKCS8 private key: %w", err)
		}
	} else {
		parsedKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			parsedKey, err = x509.ParsePKCS8PrivateKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("failed to parse PKCS1 private key: %w", err)
			}
		}
	}

	rsaKey, ok := parsedKey.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is not an RSA key")
	}

	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  pemBlockType,
		Bytes: x509.MarshalPKCS1PrivateKey(rsaKey),
	})

	return pemBytes, nil
}

// InstallationInfo GitHub App 安装信息
type InstallationInfo struct {
	ID      int64  `json:"id"`
	Account string `json:"account"`
}

// GetInstallationInfo 获取 GitHub App 安装信息
func (g *GitHubApp) GetInstallationInfo(ctx context.Context, installationID int64) (*InstallationInfo, error) {
	client, err := g.newAppClient()
	if err != nil {
		return nil, fmt.Errorf("create app client: %w", err)
	}

	installation, _, err := client.Apps.GetInstallation(ctx, installationID)
	if err != nil {
		return nil, fmt.Errorf("get installation: %w", err)
	}

	account := ""
	if installation.Account != nil {
		if login := installation.Account.GetLogin(); login != "" {
			account = login
		}
	}

	return &InstallationInfo{
		ID:      installation.GetID(),
		Account: account,
	}, nil
}

// GetInstallationAccessToken 获取安装访问令牌
func (g *GitHubApp) GetInstallationAccessToken(ctx context.Context, installationID int64) (string, error) {
	client, err := g.newInstallationClient(installationID)
	if err != nil {
		return "", fmt.Errorf("create installation client: %w", err)
	}

	token, _, err := client.Apps.CreateInstallationToken(ctx, installationID, nil)
	if err != nil {
		return "", fmt.Errorf("create installation token: %w", err)
	}

	return token.GetToken(), nil
}

// GetUserInfo 获取用户信息
func (g *GitHubApp) GetUserInfo(ctx context.Context, installationID int64) (username, email string, err error) {
	installationInfo, err := g.GetInstallationInfo(ctx, installationID)
	if err != nil {
		return "", "", fmt.Errorf("get installation info: %w", err)
	}

	username = installationInfo.Account
	if username == "" {
		username = "github-user"
	}

	email = fmt.Sprintf("%s@users.noreply.github.com", username)

	return username, email, nil
}

// newAppClient 创建 GitHub App 客户端 (用于 App 级别的 API 调用)
func (g *GitHubApp) newAppClient() (*github.Client, error) {
	itr, err := ghinstallation.NewAppsTransport(http.DefaultTransport, g.appID, g.privateKey)
	if err != nil {
		return nil, fmt.Errorf("create apps transport: %w", err)
	}
	client := github.NewClient(&http.Client{
		Transport: itr,
		Timeout:   30 * time.Second,
	})
	return client, nil
}

// newInstallationClient 创建安装客户端 (用于安装级别的 API 调用)
func (g *GitHubApp) newInstallationClient(installationID int64) (*github.Client, error) {
	itr, err := ghinstallation.New(http.DefaultTransport, g.appID, installationID, g.privateKey)
	if err != nil {
		return nil, fmt.Errorf("create installation transport: %w", err)
	}
	client := github.NewClient(&http.Client{
		Transport: itr,
		Timeout:   30 * time.Second,
	})
	return client, nil
}

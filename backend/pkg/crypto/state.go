package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// StateData State 参数数据
type StateData struct {
	UserID    uuid.UUID `json:"user_id"`
	Timestamp int64     `json:"timestamp"`
	Nonce     string    `json:"nonce"`
}

// GenerateState 生成 State 参数
func GenerateState(userID uuid.UUID, secret string) (string, error) {
	data := StateData{
		UserID:    userID,
		Timestamp: time.Now().Unix(),
		Nonce:     uuid.New().String()[:8],
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("marshal state data: %w", err)
	}

	h := hmac.New(sha256.New, []byte(secret))
	h.Write(jsonData)
	signature := h.Sum(nil)

	encodedData := base64.URLEncoding.EncodeToString(jsonData)
	encodedSig := base64.URLEncoding.EncodeToString(signature)

	return encodedData + "." + encodedSig, nil
}

// VerifyState 验证 State 参数
func VerifyState(state, secret string, maxAge int64) (*StateData, error) {
	parts := splitState(state)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid state format")
	}

	encodedData, encodedSig := parts[0], parts[1]

	jsonData, err := base64.URLEncoding.DecodeString(encodedData)
	if err != nil {
		return nil, fmt.Errorf("decode state data: %w", err)
	}

	h := hmac.New(sha256.New, []byte(secret))
	h.Write(jsonData)
	expectedSig := h.Sum(nil)

	actualSig, err := base64.URLEncoding.DecodeString(encodedSig)
	if err != nil {
		return nil, fmt.Errorf("decode signature: %w", err)
	}

	if !hmac.Equal(actualSig, expectedSig) {
		return nil, fmt.Errorf("invalid state signature")
	}

	var data StateData
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("unmarshal state data: %w", err)
	}

	if maxAge > 0 {
		age := time.Now().Unix() - data.Timestamp
		if age > maxAge {
			return nil, fmt.Errorf("state expired (age: %d seconds)", age)
		}
	}

	return &data, nil
}

func splitState(state string) []string {
	for i := len(state) - 1; i >= 0; i-- {
		if state[i] == '.' {
			return []string{state[:i], state[i+1:]}
		}
	}
	return nil
}

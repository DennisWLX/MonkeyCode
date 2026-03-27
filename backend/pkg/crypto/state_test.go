package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestGenerateAndVerifyState(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret"

	state, err := GenerateState(userID, secret)
	if err != nil {
		t.Fatalf("GenerateState failed: %v", err)
	}
	if state == "" {
		t.Fatal("Generated state is empty")
	}

	data, err := VerifyState(state, secret, 3600)
	if err != nil {
		t.Fatalf("VerifyState failed: %v", err)
	}
	if data.UserID != userID {
		t.Errorf("Expected UserID %v, got %v", userID, data.UserID)
	}
	if data.Timestamp == 0 {
		t.Error("Timestamp is zero")
	}
	if data.Nonce == "" {
		t.Error("Nonce is empty")
	}
}

func TestVerifyStateWithWrongSecret(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret"
	wrongSecret := "wrong-secret"

	state, err := GenerateState(userID, secret)
	if err != nil {
		t.Fatalf("GenerateState failed: %v", err)
	}

	_, err = VerifyState(state, wrongSecret, 3600)
	if err == nil {
		t.Fatal("Expected error with wrong secret, got nil")
	}
}

func TestVerifyStateWithExpiredState(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret"

	data := StateData{
		UserID:    userID,
		Timestamp: time.Now().Unix() - 7200, // 2 hours ago
		Nonce:     uuid.New().String()[:8],
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	h := hmac.New(sha256.New, []byte(secret))
	h.Write(jsonData)
	signature := h.Sum(nil)

	encodedData := base64.URLEncoding.EncodeToString(jsonData)
	encodedSig := base64.URLEncoding.EncodeToString(signature)
	state := encodedData + "." + encodedSig

	_, err = VerifyState(state, secret, 3600) // 1 hour max age
	if err == nil {
		t.Fatal("Expected error with expired state, got nil")
	}
}

func TestVerifyStateWithInvalidFormat(t *testing.T) {
	secret := "test-secret"

	_, err := VerifyState("invalid-state", secret, 3600)
	if err == nil {
		t.Fatal("Expected error with invalid format, got nil")
	}
}

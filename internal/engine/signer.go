package engine

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// SignaturePrefix is the standard algorithm prefix prepended to HMAC-SHA256 signatures.
const SignaturePrefix = "sha256="

// Sign generates an HMAC-SHA256 signature for the given payload using the provided secret.
func Sign(payload []byte, secret string) string {
	trimmedSecret := strings.TrimSpace(secret)
	if trimmedSecret == "" {
		return ""
	}

	// Create the hashing machine using sha-256 algorithm and the secret
	mac := hmac.New(sha256.New, []byte(trimmedSecret))

	// Feed the payload into the hashing machine
	mac.Write(payload)

	// Get the hash and convert it to a hex string
	hash := hex.EncodeToString(mac.Sum(nil))

	return SignaturePrefix + hash
}

// Verify checks whether a provided signature matches the expected HMAC-SHA256 signature
// of the payload and secret. 
// It uses constant-time comparison (hmac.Equal) to prevent timing side-channel attacks.

func Verify(payload []byte, secret string, signature string) bool {
	trimmedSecret := strings.TrimSpace(secret)
	trimmedSig := strings.TrimSpace(signature)

	if trimmedSecret == "" || trimmedSig == "" {
		return false
	}

	if !strings.HasPrefix(trimmedSig, SignaturePrefix) {
		return false
	}

	expected := Sign(payload, trimmedSecret)
	if expected == "" {
		return false
	}

	return hmac.Equal([]byte(expected), []byte(trimmedSig))
}

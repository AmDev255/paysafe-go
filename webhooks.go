package paysafe

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

const (
	SignatureHeader         = "PaySafe-Signature"
	DefaultToleranceSeconds = 300
	maxSignatureHeaderBytes = 4096
	maxWebhookPayloadBytes  = 1 << 20
)

// VerifyOptions tune signature checks.
type VerifyOptions struct {
	// Seconds the delivery timestamp may differ from now; 0 means DefaultToleranceSeconds.
	Tolerance int
	// Unix seconds to compare against; 0 means the current time.
	Now int64
}

func hmacHex(secret string, timestamp int64, payload []byte) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte(strconv.FormatInt(timestamp, 10) + "."))
	m.Write(payload)
	return hex.EncodeToString(m.Sum(nil))
}

// Sign builds a PaySafe-Signature value. Intended for tests.
func Sign(secret string, payload []byte, timestamp int64) string {
	return "t=" + strconv.FormatInt(timestamp, 10) + ",v1=" + hmacHex(secret, timestamp, payload)
}

func checkSignature(secrets []string, payload []byte, header string, opts *VerifyOptions) error {
	if strings.TrimSpace(header) == "" {
		return &SignatureError{Reason: SignatureMissingHeader}
	}
	if len(header) > maxSignatureHeaderBytes || len(payload) > maxWebhookPayloadBytes {
		return &SignatureError{Reason: SignatureMalformedHeader}
	}
	var timestamp string
	var signatures []string
	for _, part := range strings.Split(header, ",") {
		key, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		switch key {
		case "t":
			timestamp = value
		case "v1":
			signatures = append(signatures, strings.ToLower(strings.TrimSpace(value)))
		}
	}
	t, err := strconv.ParseInt(timestamp, 10, 64)
	if timestamp == "" || err != nil || t < 0 || len(signatures) == 0 {
		return &SignatureError{Reason: SignatureMalformedHeader}
	}
	tolerance := int64(DefaultToleranceSeconds)
	now := time.Now().Unix()
	if opts != nil {
		if opts.Tolerance > 0 {
			tolerance = int64(opts.Tolerance)
		}
		if opts.Now > 0 {
			now = opts.Now
		}
	}
	if d := now - t; d > tolerance || -d > tolerance {
		return &SignatureError{Reason: SignatureTimestampOutsideTolerance}
	}
	for _, secret := range secrets {
		expected := []byte(hmacHex(secret, t, payload))
		for _, sig := range signatures {
			if subtle.ConstantTimeCompare(expected, []byte(sig)) == 1 {
				return nil
			}
		}
	}
	return &SignatureError{Reason: SignatureMismatch}
}

// VerifySignature reports whether payload (the exact bytes received) was signed by one of the secrets.
func VerifySignature(secret string, payload []byte, header string, opts *VerifyOptions) bool {
	return checkSignature([]string{secret}, payload, header, opts) == nil
}

// VerifySignatureAny accepts several secrets, for rotations on your side.
func VerifySignatureAny(secrets []string, payload []byte, header string, opts *VerifyOptions) bool {
	return checkSignature(secrets, payload, header, opts) == nil
}

// ConstructEvent verifies the delivery and decodes the event. The returned error is a *SignatureError
// with a Reason, or an *InvalidRequestError for an unparsable payload.
func ConstructEvent(secret string, payload []byte, header string, opts *VerifyOptions) (*Event, error) {
	return ConstructEventAny([]string{secret}, payload, header, opts)
}

// ConstructEventAny is ConstructEvent with several candidate secrets.
func ConstructEventAny(secrets []string, payload []byte, header string, opts *VerifyOptions) (*Event, error) {
	if err := checkSignature(secrets, payload, header, opts); err != nil {
		return nil, err
	}
	var event Event
	if err := json.Unmarshal(payload, &event); err != nil || event.ID == "" || event.Type == "" {
		return nil, &SignatureError{Reason: SignatureInvalidPayload}
	}
	return &event, nil
}

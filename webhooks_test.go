package paysafe_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	paysafe "github.com/AmDev255/paysafe-go"
)

type vector struct {
	Secret    string `json:"secret"`
	Timestamp int64  `json:"timestamp"`
	Body      string `json:"body"`
	Signature string `json:"signature"`
	Header    string `json:"header"`
	Legacy    struct {
		Secret    string `json:"secret"`
		Signature string `json:"signature"`
	} `json:"legacy"`
	Rotation struct {
		Header string `json:"header"`
	} `json:"rotation"`
	SecondSecret struct {
		Secret    string `json:"secret"`
		Signature string `json:"signature"`
	} `json:"secondSecret"`
	NowValid     int64  `json:"nowValid"`
	NowExpired   int64  `json:"nowExpired"`
	NowFuture    int64  `json:"nowFuture"`
	TamperedBody string `json:"tamperedBody"`
}

func loadVector(t testing.TB) (vector, *paysafe.VerifyOptions) {
	var v vector
	loadFixture(t, "webhook-vector", &v)
	return v, &paysafe.VerifyOptions{Now: v.NowValid}
}

func TestWebhookVector(t *testing.T) {
	v, at := loadVector(t)
	if !paysafe.VerifySignature(v.Secret, []byte(v.Body), v.Header, at) {
		t.Fatal("documented vector does not verify")
	}
	legacy := fmt.Sprintf("t=%d,v1=%s", v.Timestamp, v.Legacy.Signature)
	if !paysafe.VerifySignature(v.Legacy.Secret, []byte(v.Body), legacy, at) || paysafe.VerifySignature(v.Secret, []byte(v.Body), legacy, at) {
		t.Fatal("legacy vector")
	}
	for name, ok := range map[string]bool{
		"wrong secret":  paysafe.VerifySignature("paysafe_webhook_test_wrong", []byte(v.Body), v.Header, at),
		"tampered":      paysafe.VerifySignature(v.Secret, []byte(v.TamperedBody), v.Header, at),
		"missing":       paysafe.VerifySignature(v.Secret, []byte(v.Body), "", at),
		"no t":          paysafe.VerifySignature(v.Secret, []byte(v.Body), "v1="+v.Signature, at),
		"bad t":         paysafe.VerifySignature(v.Secret, []byte(v.Body), "t=abc,v1="+v.Signature, at),
		"no v1":         paysafe.VerifySignature(v.Secret, []byte(v.Body), fmt.Sprintf("t=%d", v.Timestamp), at),
		"expired":       paysafe.VerifySignature(v.Secret, []byte(v.Body), v.Header, &paysafe.VerifyOptions{Now: v.NowExpired}),
		"future":        paysafe.VerifySignature(v.Secret, []byte(v.Body), v.Header, &paysafe.VerifyOptions{Now: v.NowFuture}),
		"wrong of list": paysafe.VerifySignatureAny([]string{v.Secret}, []byte(v.Body), fmt.Sprintf("t=%d,v1=%s", v.Timestamp, v.SecondSecret.Signature), at),
	} {
		if ok {
			t.Fatalf("%s accepted", name)
		}
	}
	if !paysafe.VerifySignature(v.Secret, []byte(v.Body), v.Header, &paysafe.VerifyOptions{Now: v.NowExpired, Tolerance: 1000}) {
		t.Fatal("tolerance override")
	}
	if !paysafe.VerifySignature(v.Secret, []byte(v.Body), v.Rotation.Header, at) {
		t.Fatal("rotation header")
	}
	if !paysafe.VerifySignatureAny([]string{v.Secret, v.SecondSecret.Secret}, []byte(v.Body), fmt.Sprintf("t=%d,v1=%s", v.Timestamp, v.SecondSecret.Signature), at) {
		t.Fatal("secret list")
	}
}

func TestConstructEvent(t *testing.T) {
	v, at := loadVector(t)
	event, err := paysafe.ConstructEvent(v.Secret, []byte(v.Body), v.Header, at)
	if err != nil || event.ID != "evt_01J7A9R5VKM4N5P6R7S8T9V0B6" || event.Type != "payment.succeeded" || event.Livemode {
		t.Fatalf("%v %v", event, err)
	}
	p, ok := event.Payment()
	if !ok || p.ID != "pay_01J7A9Q0K3M4N5P6R7S8T9V0W1" || p.Status != "succeeded" {
		t.Fatalf("payment %v %v", p, ok)
	}
	if _, ok := event.Refund(); ok {
		t.Fatal("refund decoded from a payment event")
	}

	_, err = paysafe.ConstructEvent(v.Secret, []byte(v.TamperedBody), v.Header, at)
	var se *paysafe.SignatureError
	if !errors.As(err, &se) || se.Reason != paysafe.SignatureMismatch || strings.Contains(err.Error(), v.Secret) || strings.Contains(err.Error(), v.Signature) {
		t.Fatalf("mismatch %v", err)
	}
	_, err = paysafe.ConstructEvent(v.Secret, []byte(v.Body), v.Header, &paysafe.VerifyOptions{Now: v.NowExpired})
	if !errors.As(err, &se) || se.Reason != paysafe.SignatureTimestampOutsideTolerance {
		t.Fatalf("expired %v", err)
	}
	body := []byte("[1,2,3]")
	_, err = paysafe.ConstructEvent(v.Secret, body, paysafe.Sign(v.Secret, body, v.Timestamp), at)
	if !errors.As(err, &se) || se.Reason != paysafe.SignatureInvalidPayload {
		t.Fatalf("payload %v", err)
	}
	if paysafe.Sign(v.Secret, []byte(v.Body), v.Timestamp) != v.Header {
		t.Fatal("Sign does not reproduce the header")
	}
}

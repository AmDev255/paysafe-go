// Minimal webhook receiver on net/http. Verifies the signature before anything else.
// Run:  PAYSAFE_WEBHOOK_SECRET=paysafe_webhook_... go run ./examples/verify-webhook
// Self-test without a secret: go run ./examples/verify-webhook
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"

	paysafe "github.com/AmDev255/paysafe-go"
)

func main() {
	secret := os.Getenv("PAYSAFE_WEBHOOK_SECRET")
	if secret == "" {
		selfTest()
		return
	}
	var mu sync.Mutex
	seen := map[string]bool{}

	http.HandleFunc("/paysafe/webhook", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		rawBody, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		event, err := paysafe.ConstructEvent(secret, rawBody, r.Header.Get(paysafe.SignatureHeader), nil)
		if err != nil {
			var se *paysafe.SignatureError
			if errors.As(err, &se) {
				http.Error(w, "invalid signature: "+se.Reason, http.StatusBadRequest)
				return
			}
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		mu.Lock()
		dup := seen[event.ID]
		seen[event.ID] = true
		mu.Unlock()
		if dup {
			fmt.Fprint(w, "duplicate")
			return
		}
		switch event.Type {
		case "payment.succeeded":
			if p, ok := event.Payment(); ok {
				fmt.Printf("paid %s %d\n", p.ID, p.Amount)
			}
		case "payment.failed", "payment.expired":
			fmt.Printf("not paid: %s\n", event.Type)
		default:
			fmt.Printf("ignored %s\n", event.Type)
		}
		fmt.Fprint(w, "ok")
	})
	fmt.Println("Listening on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func selfTest() {
	data, err := os.ReadFile("tests/fixtures/webhook-vector.json")
	if err != nil {
		fmt.Fprintln(os.Stderr, "run from the repository root:", err)
		os.Exit(1)
	}
	var v struct {
		Secret   string `json:"secret"`
		Body     string `json:"body"`
		Header   string `json:"header"`
		NowValid int64  `json:"nowValid"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	event, err := paysafe.ConstructEvent(v.Secret, []byte(v.Body), v.Header, &paysafe.VerifyOptions{Now: v.NowValid})
	if err != nil {
		fmt.Fprintln(os.Stderr, "self-test failed:", err)
		os.Exit(1)
	}
	p, _ := event.Payment()
	fmt.Printf("Self-test ok: %s %s\n", event.Type, p.ID)
}

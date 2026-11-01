package paysafe_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	paysafe "github.com/AmDev255/paysafe-go"
)

const (
	testKey   = "paysafe_test_9f2Kx7Qm3Rt8Vw2Yz5Ab1Cd4Ef6Gh9Jk"
	requestID = "req_01J7A9Q0K3M4N5P6R7S8T9V0W1"
)

type operationFixture struct {
	OperationID  string            `json:"operationId"`
	Method       string            `json:"method"`
	Path         string            `json:"path"`
	ResolvedPath string            `json:"resolvedPath"`
	Query        map[string]any    `json:"query"`
	Headers      map[string]string `json:"headers"`
	Body         json.RawMessage   `json:"body"`
	MinimalBody  json.RawMessage   `json:"minimalBody"`
	Responses    map[string]struct {
		Headers map[string]string `json:"headers"`
		Body    json.RawMessage   `json:"body"`
		Code    string            `json:"code"`
	} `json:"responses"`
}

func loadFixture(t testing.TB, name string, v any) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("tests", "fixtures", name+".json"))
	if err != nil {
		t.Fatalf("fixture %s: %v", name, err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("fixture %s: %v", name, err)
	}
}

func loadOperation(t testing.TB, id string) operationFixture {
	var f operationFixture
	loadFixture(t, id, &f)
	return f
}

func successOf(f operationFixture) (int, json.RawMessage, map[string]string) {
	for code, r := range f.Responses {
		if strings.HasPrefix(code, "2") {
			n := 0
			for _, ch := range code {
				n = n*10 + int(ch-'0')
			}
			return n, r.Body, r.Headers
		}
	}
	panic("no success response")
}

type queued struct {
	resp *http.Response
	err  error
	hang bool
}

// fakeTransport answers from a queue and records every request.
type fakeTransport struct {
	queue    []queued
	Requests []*http.Request
	Bodies   [][]byte
}

func (f *fakeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var body []byte
	if req.Body != nil {
		body, _ = io.ReadAll(req.Body)
	}
	f.Requests = append(f.Requests, req)
	f.Bodies = append(f.Bodies, body)
	if len(f.queue) == 0 {
		return nil, errors.New("no queued response for " + req.Method + " " + req.URL.String())
	}
	next := f.queue[0]
	f.queue = f.queue[1:]
	if next.hang {
		<-req.Context().Done()
		return nil, req.Context().Err()
	}
	if next.err != nil {
		return nil, next.err
	}
	next.resp.Request = req
	return next.resp, nil
}

func (f *fakeTransport) JSON(status int, body any, headers map[string]string) *fakeTransport {
	var raw []byte
	switch b := body.(type) {
	case nil:
	case json.RawMessage:
		raw = b
	case string:
		raw = []byte(b)
	default:
		raw, _ = json.Marshal(b)
	}
	h := http.Header{}
	for k, v := range headers {
		h.Set(k, v)
	}
	if h.Get("X-Request-ID") == "" {
		h.Set("X-Request-ID", requestID)
	}
	if len(raw) > 0 && h.Get("Content-Type") == "" {
		h.Set("Content-Type", "application/json")
	}
	resp := &http.Response{StatusCode: status, Header: h, Body: io.NopCloser(bytes.NewReader(raw)), ContentLength: int64(len(raw))}
	f.queue = append(f.queue, queued{resp: resp})
	return f
}

func (f *fakeTransport) Raw(status int, body string, headers map[string]string) *fakeTransport {
	h := http.Header{}
	for k, v := range headers {
		h.Set(k, v)
	}
	length := int64(len(body))
	if cl := h.Get("Content-Length"); cl != "" {
		fmt.Sscan(cl, &length)
	}
	f.queue = append(f.queue, queued{resp: &http.Response{StatusCode: status, Header: h, Body: io.NopCloser(strings.NewReader(body)), ContentLength: length}})
	return f
}

func (f *fakeTransport) Err(err error) *fakeTransport {
	f.queue = append(f.queue, queued{err: err})
	return f
}

func (f *fakeTransport) Hang() *fakeTransport {
	f.queue = append(f.queue, queued{hang: true})
	return f
}

func (f *fakeTransport) Last() *http.Request { return f.Requests[len(f.Requests)-1] }
func (f *fakeTransport) LastBody() []byte    { return f.Bodies[len(f.Bodies)-1] }
func (f *fakeTransport) Pending() int        { return len(f.queue) }

type sleeps struct{ Durations []time.Duration }

func (s *sleeps) Sleep(_ context.Context, d time.Duration) error {
	s.Durations = append(s.Durations, d)
	return nil
}

func (s *sleeps) Total() time.Duration {
	var t time.Duration
	for _, d := range s.Durations {
		t += d
	}
	return t
}

func newClient(t testing.TB, maxRetries int, opts ...paysafe.Option) (*paysafe.Client, *fakeTransport, *sleeps) {
	t.Helper()
	ft := &fakeTransport{}
	sl := &sleeps{}
	all := append([]paysafe.Option{
		paysafe.WithBaseURL("https://api.test.invalid"),
		paysafe.WithMaxRetries(maxRetries),
		paysafe.WithHTTPClient(&http.Client{Transport: ft}),
		paysafe.WithTestSleep(sl.Sleep),
		paysafe.WithTestRand(func() float64 { return 0.5 }),
	}, opts...)
	c, err := paysafe.NewClient(testKey, all...)
	if err != nil {
		t.Fatal(err)
	}
	return c, ft, sl
}

func mustJSON(t testing.TB, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

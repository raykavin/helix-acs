package auth

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/raykavin/helix-acs/packages/logger"
)

type mockLogger struct {
	t *testing.T
}

// Debug implements [logger.Logger].
func (m *mockLogger) Debug(args ...any) {

}

// Debugf implements [logger.Logger].
func (m *mockLogger) Debugf(format string, args ...any) {

}

// Error implements [logger.Logger].
func (m *mockLogger) Error(args ...any) {

}

// Errorf implements [logger.Logger].
func (m *mockLogger) Errorf(format string, args ...any) {

}

// Fatal implements [logger.Logger].
func (m *mockLogger) Fatal(args ...any) {

}

// Fatalf implements [logger.Logger].
func (m *mockLogger) Fatalf(format string, args ...any) {

}

// Info implements [logger.Logger].
func (m *mockLogger) Info(args ...any) {

}

// Infof implements [logger.Logger].
func (m *mockLogger) Infof(format string, args ...any) {

}

// Panic implements [logger.Logger].
func (m *mockLogger) Panic(args ...any) {

}

// Panicf implements [logger.Logger].
func (m *mockLogger) Panicf(format string, args ...any) {

}

// Print implements [logger.Logger].
func (m *mockLogger) Print(args ...any) {

}

// Printf implements [logger.Logger].
func (m *mockLogger) Printf(format string, args ...any) {

}

// Warn implements [logger.Logger].
func (m *mockLogger) Warn(args ...any) {

}

// Warnf implements [logger.Logger].
func (m *mockLogger) Warnf(format string, args ...any) {

}

// WithError implements [logger.Logger].
func (m *mockLogger) WithError(err error) logger.Logger {
	return m
}

// WithField implements [logger.Logger].
func (m *mockLogger) WithField(key string, value any) logger.Logger {
	return m
}

// WithFields implements [logger.Logger].
func (m *mockLogger) WithFields(fields map[string]any) logger.Logger {
	return m
}

func testMD5Hex(s string) string {
	h := md5.New()
	_, _ = h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func parseTestChallenge(s string) map[string]string {
	params := make(map[string]string)
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		idx := strings.IndexByte(part, '=')
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(part[:idx])
		val := strings.TrimSpace(part[idx+1:])
		if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
			val = val[1 : len(val)-1]
		}
		params[key] = val
	}
	return params
}

func TestDigestAuthFullFlow(t *testing.T) {
	log := &mockLogger{t: t}

	realm := "ACS"
	username := "acs"
	password := "acs123"
	method := "POST"
	uri := "/acs"

	da := NewDigestAuth(log, realm, username, password)

	// Step 1: Unauthenticated request → get 401 challenge
	req1 := httptest.NewRequest(method, uri, nil)
	rec1 := httptest.NewRecorder()

	var nonce string
	handler := da.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec1.Code)
	}

	challenge := rec1.Header().Get("WWW-Authenticate")
	t.Logf("Challenge: %s", challenge)

	if !strings.HasPrefix(challenge, "Digest ") {
		t.Fatalf("unexpected challenge: %s", challenge)
	}

	parsed := parseTestChallenge(challenge[7:])
	nonce = parsed["nonce"]
	parsedRealm := parsed["realm"]

	t.Logf("Parsed realm: %q", parsedRealm)
	t.Logf("Parsed nonce: %q", nonce)

	// Step 2: Compute digest
	cnonce := "testcnon"
	nc := "00000001"
	qop := "auth"

	ha1 := testMD5Hex(username + ":" + parsedRealm + ":" + password)
	ha2 := testMD5Hex(method + ":" + uri)
	digestResponse := testMD5Hex(ha1 + ":" + nonce + ":" + nc + ":" + cnonce + ":" + qop + ":" + ha2)

	t.Logf("ha1=%s ha2=%s response=%s", ha1, ha2, digestResponse)

	authHeader := fmt.Sprintf(
		`Digest username=%q, realm=%q, nonce=%q, uri=%q, algorithm=MD5, qop=auth, nc=%s, cnonce=%q, response=%q`,
		username, parsedRealm, nonce, uri, nc, cnonce, digestResponse,
	)
	t.Logf("Authorization: %s", authHeader)

	req2 := httptest.NewRequest(method, uri, nil)
	req2.Header.Set("Authorization", authHeader)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Errorf("expected 200, got %d (auth failed)", rec2.Code)
	} else {
		t.Log("Auth succeeded!")
	}
}

func TestParseDigestHeaderValues(t *testing.T) {
	input := `username="acs", realm="ACS", nonce="f47ac10b-58cc-4372-a567-0e02b2c3d479", uri="/acs", algorithm=MD5, qop=auth, nc=00000001, cnonce="f47ac10b", response="d8ec5e2f5b823e4a21e58e5e98f79534"`

	params := parseDigestHeader(input)

	tests := []struct{ key, want string }{
		{"username", "acs"},
		{"realm", "ACS"},
		{"nonce", "f47ac10b-58cc-4372-a567-0e02b2c3d479"},
		{"uri", "/acs"},
		{"algorithm", "MD5"},
		{"qop", "auth"},
		{"nc", "00000001"},
		{"cnonce", "f47ac10b"},
		{"response", "d8ec5e2f5b823e4a21e58e5e98f79534"},
	}
	for _, tt := range tests {
		got := params[tt.key]
		if got != tt.want {
			t.Errorf("params[%q] = %q, want %q", tt.key, got, tt.want)
		}
	}
}

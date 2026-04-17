package auth

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/raykavin/helix-acs/packages/logger"
)

const (
	digestNonceTTL    = 5 * time.Minute
	digestMaxNonceUse = 1000
)

// nonceEntry tracks a generated nonce so stale ones can be pruned.
type nonceEntry struct {
	created time.Time
	used    int
}

// DigestAuth implements RFC 2617 HTTP Digest Authentication. It is used to
// authenticate TR-069 CPE devices hitting the CWMP endpoint.
type DigestAuth struct {
	realm    string
	username string
	password string
	nonces   map[string]nonceEntry
	mu       sync.Mutex
	log      logger.Logger
}

// NewDigestAuth returns a DigestAuth configured for the given realm and
// expected CPE credentials.
func NewDigestAuth(log logger.Logger, realm, username, password string) *DigestAuth {
	return &DigestAuth{
		realm:    realm,
		username: username,
		password: password,
		log:      log,
		nonces:   make(map[string]nonceEntry),
	}
}

// Middleware wraps next with Digest authentication enforcement. Unauthenticated
// requests receive a 401 with a WWW-Authenticate challenge header.
func (d *DigestAuth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		d.cleanStaleNonces()

		_, ok := d.ValidateRequest(r)
		if !ok {
			nonce := d.newNonce()
			w.Header().Set("WWW-Authenticate", fmt.Sprintf(
				`Digest realm=%q, nonce=%q, qop="auth", algorithm="MD5"`,
				d.realm, nonce,
			))
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("Unauthorized"))
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ValidateRequest inspects the Authorization header and returns the
// authenticated username plus true when the digest is valid. It returns
// ("", false) for any failure.
func (d *DigestAuth) ValidateRequest(r *http.Request) (string, bool) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", false
	}

	if !strings.HasPrefix(authHeader, "Digest ") {
		return "", false
	}

	params := parseDigestHeader(authHeader[7:])

	username := params["username"]
	realm := params["realm"]
	nonce := params["nonce"]
	uri := params["uri"]
	qop := params["qop"]
	nc := params["nc"]
	cnonce := params["cnonce"]
	response := params["response"]

	if username == "" || realm == "" || nonce == "" || uri == "" || response == "" {
		d.log.WithField("remote", r.RemoteAddr).Debug("Digest auth: Missing required parameters")
		return "", false
	}

	// Reject unknown nonces (replay / forgery).
	d.mu.Lock()
	entry, known := d.nonces[nonce]
	if known {
		entry.used++
		d.nonces[nonce] = entry
	}
	d.mu.Unlock()

	if !known {
		d.log.WithField("nonce", nonce).Debug("Digest auth: Unknown or expired nonce")
		return "", false
	}

	if entry.used > digestMaxNonceUse {
		d.log.WithField("nonce", nonce).WithField("used", entry.used).Warn("Digest auth: Nonce exceeded max use count")
		return "", false
	}

	// Compute expected response.
	ha1 := md5hex(username + ":" + realm + ":" + d.password)
	ha2 := md5hex(r.Method + ":" + uri)

	var expected string
	if qop == "auth" || qop == "auth-int" {
		expected = md5hex(ha1 + ":" + nonce + ":" + nc + ":" + cnonce + ":" + qop + ":" + ha2)
	} else {
		// Legacy / no-qop mode (RFC 2069 compatible).
		expected = md5hex(ha1 + ":" + nonce + ":" + ha2)
	}

	if !strings.EqualFold(response, expected) {
		d.log.
			WithField("remote", r.RemoteAddr).
			WithField("username", username).
			Debug("Digest auth: Response mismatch")
		return "", false
	}

	return username, true
}

// newNonce generates a UUID nonce, stores it, and returns it.
func (d *DigestAuth) newNonce() string {
	nonce := uuid.New().String()
	d.mu.Lock()
	d.nonces[nonce] = nonceEntry{created: time.Now()}
	d.mu.Unlock()
	return nonce
}

// cleanStaleNonces removes nonces that are older than digestNonceTTL.
func (d *DigestAuth) cleanStaleNonces() {
	cutoff := time.Now().Add(-digestNonceTTL)
	d.mu.Lock()
	defer d.mu.Unlock()
	for n, e := range d.nonces {
		if e.created.Before(cutoff) {
			delete(d.nonces, n)
		}
	}
}

// md5hex returns the lower-case hex-encoded MD5 of s.
func md5hex(s string) string {
	h := md5.New() //nolint:gosec
	_, _ = h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// parseDigestHeader parses the comma-separated key=value (or key="value")
// pairs that follow the "Digest " prefix in an Authorization header.
func parseDigestHeader(s string) map[string]string {
	params := make(map[string]string)
	for part := range strings.SplitSeq(s, ",") {
		part = strings.TrimSpace(part)
		before, after, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}

		key := strings.TrimSpace(before)
		val := strings.TrimSpace(after)
		// Strip surrounding quotes if present.
		if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
			val = val[1 : len(val)-1]
		}
		params[key] = val
	}
	return params
}

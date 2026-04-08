// cpe_simulator simulates a TR-069 CPE (router/modem) connecting to the Helix ACS.
//
// It replicates the minimal TR-069 session lifecycle:
//  1. POST Inform → receive InformResponse
//  2. POST empty body → receive 204 (session end)
//
// Digest Authentication (RFC 2617) is handled automatically: the simulator
// detects the 401 challenge, computes the MD5 response, and retries.
//
// Usage:
//
//	go run ./examples/cpe_simulator \
//	  -acs http://localhost:7547/acs \
//	  -username acs \
//	  -password acs123 \
//	  -serial SN-SIMULATOR-001 \
//	  -manufacturer Intelbras \
//	  -oui 001122 \
//	  -product WiFiRouter
package main

import (
	"bytes"
	"crypto/md5"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ── CLI flags ────────────────────────────────────────────────────────────────

var (
	flagACS          = flag.String("acs", "http://localhost:7547/acs", "ACS URL")
	flagUsername     = flag.String("username", "acs", "ACS username (Digest)")
	flagPassword     = flag.String("password", "acs123", "ACS password (Digest)")
	flagSerial       = flag.String("serial", "SN-SIMULATOR-001", "CPE serial number")
	flagManufacturer = flag.String("manufacturer", "Intelbras", "CPE manufacturer")
	flagOUI          = flag.String("oui", "001122", "CPE OUI")
	flagProduct      = flag.String("product", "WiFiRouter", "CPE product class")
	flagSWVersion    = flag.String("sw", "1.2.3", "CPE software version")
	flagHWVersion    = flag.String("hw", "1.0", "CPE hardware version")
	flagWANIP        = flag.String("wanip", "192.168.1.1", "CPE WAN IP address")
	flagInterval     = flag.Int("interval", 0, "Inform interval in seconds (0 = single shot)")
)

// ── Main ─────────────────────────────────────────────────────────────────────

func main() {
	flag.Parse()

	logf("CPE Simulator starting")
	logf("  ACS URL    : %s", *flagACS)
	logf("  Serial     : %s", *flagSerial)
	logf("  Manufacturer: %s  OUI: %s  Product: %s", *flagManufacturer, *flagOUI, *flagProduct)

	if *flagInterval > 0 {
		logf("  Mode: periodic, every %ds", *flagInterval)
		for {
			runSession()
			logf("Waiting %ds before next Inform...", *flagInterval)
			time.Sleep(time.Duration(*flagInterval) * time.Second)
		}
	} else {
		logf("  Mode: single-shot")
		runSession()
	}
}

// ── Session ───────────────────────────────────────────────────────────────────

func runSession() {
	sessionID := uuid.NewString()
	logf("\n=== Session %s ===", sessionID[:8])

	// Step 1: Send Inform
	informBody := buildInform(sessionID)
	logf("[1/2] Sending Inform...")

	resp, body, cookies, err := doRequestWithDigest("POST", *flagACS, *flagUsername, *flagPassword, []byte(informBody), nil)
	if err != nil {
		logf("ERROR: Inform failed: %v", err)
		os.Exit(1)
	}
	logf("      → HTTP %d", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		logf("      Response body: %s", string(body))
		os.Exit(1)
	}
	if strings.Contains(string(body), "InformResponse") {
		logf("      ✓ InformResponse received")
	} else {
		logf("      WARNING: unexpected response body: %s", string(body))
	}

	// Step 2: Send empty body to end session
	logf("[2/2] Sending empty body (session end)...")
	resp2, _, _, err := doRequestWithDigest("POST", *flagACS, *flagUsername, *flagPassword, []byte{}, cookies)
	if err != nil {
		logf("ERROR: session-end POST failed: %v", err)
		os.Exit(1)
	}
	logf("      → HTTP %d", resp2.StatusCode)
	if resp2.StatusCode == http.StatusNoContent {
		logf("      ✓ Session closed gracefully (204 No Content)")
	} else {
		logf("      WARNING: expected 204, got %d", resp2.StatusCode)
	}

	logf("=== Session complete ===")
}

// ── HTTP + Digest Auth ────────────────────────────────────────────────────────

// doRequestWithDigest sends a POST and transparently handles the 401 → Digest
// retry cycle. Returns the final response, body bytes, and cookies to reuse.
func doRequestWithDigest(method, url, username, password string, body []byte, cookies []*http.Cookie) (*http.Response, []byte, []*http.Cookie, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	// First attempt (no auth header yet – or pre-authenticated if cookies set).
	req, err := newReq(method, url, body)
	if err != nil {
		return nil, nil, nil, err
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		b, _ := io.ReadAll(resp.Body)
		return resp, b, resp.Cookies(), nil
	}

	// Parse the Digest challenge from WWW-Authenticate.
	challenge := resp.Header.Get("WWW-Authenticate")
	if !strings.HasPrefix(challenge, "Digest ") {
		return nil, nil, nil, fmt.Errorf("unexpected auth scheme: %s", challenge)
	}
	logf("      ← 401 Digest challenge received, computing response...")

	params := parseDigestChallenge(challenge[7:])
	realm := params["realm"]
	nonce := params["nonce"]
	qop := params["qop"]
	algorithm := params["algorithm"]
	if algorithm == "" {
		algorithm = "MD5"
	}

	cnonce := uuid.NewString()[:8]
	nc := "00000001"
	uri := extractPath(url)

	ha1 := md5hex(username + ":" + realm + ":" + password)
	ha2 := md5hex(method + ":" + uri)

	var digestResp string
	if strings.Contains(qop, "auth") {
		digestResp = md5hex(ha1 + ":" + nonce + ":" + nc + ":" + cnonce + ":auth:" + ha2)
	} else {
		digestResp = md5hex(ha1 + ":" + nonce + ":" + ha2)
	}

	authHeader := fmt.Sprintf(
		`Digest username=%q, realm=%q, nonce=%q, uri=%q, algorithm=%s, qop=auth, nc=%s, cnonce=%q, response=%q`,
		username, realm, nonce, uri, algorithm, nc, cnonce, digestResp,
	)

	// Retry with the Digest Authorization header.
	req2, err := newReq(method, url, body)
	if err != nil {
		return nil, nil, nil, err
	}
	req2.Header.Set("Authorization", authHeader)
	for _, c := range cookies {
		req2.AddCookie(c)
	}
	// Carry over session cookie from the 401 response.
	for _, c := range resp.Cookies() {
		req2.AddCookie(c)
	}

	resp2, err := client.Do(req2)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("http (digest retry): %w", err)
	}
	defer resp2.Body.Close()

	b, _ := io.ReadAll(resp2.Body)
	return resp2, b, resp2.Cookies(), nil
}

func newReq(method, url string, body []byte) (*http.Request, error) {
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	}
	req.Header.Set("User-Agent", "CPE-Simulator/1.0")
	return req, nil
}

// ── SOAP builder ──────────────────────────────────────────────────────────────

func buildInform(sessionID string) string {
	now := time.Now().UTC().Format(time.RFC3339)
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope>
  <soap:Header>
    <cwmp:ID mustUnderstand="1">%s</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:Inform>
      <DeviceId>
        <Manufacturer>%s</Manufacturer>
        <OUI>%s</OUI>
        <ProductClass>%s</ProductClass>
        <SerialNumber>%s</SerialNumber>
      </DeviceId>
      <Event>
        <EventStruct>
          <EventCode>0 BOOTSTRAP</EventCode>
          <CommandKey></CommandKey>
        </EventStruct>
      </Event>
      <MaxEnvelopes>1</MaxEnvelopes>
      <CurrentTime>%s</CurrentTime>
      <RetryCount>0</RetryCount>
      <ParameterList>
        <ParameterValueStruct>
          <Name>Device.DeviceInfo.Manufacturer</Name>
          <Value xsi:type="xsd:string">%s</Value>
        </ParameterValueStruct>
        <ParameterValueStruct>
          <Name>Device.DeviceInfo.SerialNumber</Name>
          <Value xsi:type="xsd:string">%s</Value>
        </ParameterValueStruct>
        <ParameterValueStruct>
          <Name>Device.DeviceInfo.SoftwareVersion</Name>
          <Value xsi:type="xsd:string">%s</Value>
        </ParameterValueStruct>
        <ParameterValueStruct>
          <Name>Device.DeviceInfo.HardwareVersion</Name>
          <Value xsi:type="xsd:string">%s</Value>
        </ParameterValueStruct>
        <ParameterValueStruct>
          <Name>Device.ManagementServer.URL</Name>
          <Value xsi:type="xsd:string">%s</Value>
        </ParameterValueStruct>
        <ParameterValueStruct>
          <Name>Device.ManagementServer.Username</Name>
          <Value xsi:type="xsd:string">%s</Value>
        </ParameterValueStruct>
        <ParameterValueStruct>
          <Name>Device.IP.Interface.1.IPv4Address.1.IPAddress</Name>
          <Value xsi:type="xsd:string">%s</Value>
        </ParameterValueStruct>
      </ParameterList>
    </cwmp:Inform>
  </soap:Body>
</soap:Envelope>`,
		sessionID,
		*flagManufacturer, *flagOUI, *flagProduct, *flagSerial,
		now,
		*flagManufacturer, *flagSerial,
		*flagSWVersion, *flagHWVersion,
		*flagACS, *flagUsername,
		*flagWANIP,
	)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func md5hex(s string) string {
	h := md5.New() //nolint:gosec
	_, _ = h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func parseDigestChallenge(s string) map[string]string {
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

func extractPath(rawURL string) string {
	// Strips scheme+host, returns "/path?query".
	for _, prefix := range []string{"http://", "https://"} {
		if strings.HasPrefix(rawURL, prefix) {
			rest := rawURL[len(prefix):]
			slash := strings.IndexByte(rest, '/')
			if slash >= 0 {
				return rest[slash:]
			}
			return "/"
		}
	}
	return rawURL
}

func logf(format string, args ...any) {
	fmt.Fprintf(os.Stdout, "[%s] %s\n", time.Now().Format("15:04:05"), fmt.Sprintf(format, args...))
}

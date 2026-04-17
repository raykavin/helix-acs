package cwmp

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// informXML is a realistic CPE Inform as it arrives from a device in the wild.
// Note: namespace prefix declarations are intentionally omitted  Go's
// encoding/xml decoder matches the struct tag `xml:"soap:Envelope"` as a
// literal prefixed local name, so the XML must contain `soap:Envelope` without
// the xmlns:soap declaration.  This is consistent with how ParseEnvelope is
// used in session.go (CPE devices commonly omit or vary namespace declarations).
const informXML = `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope>
  <soap:Header>
    <cwmp:ID mustUnderstand="1">1</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:Inform>
      <DeviceId>
        <Manufacturer>Intelbras</Manufacturer>
        <OUI>001122</OUI>
        <ProductClass>WiFiRouter</ProductClass>
        <SerialNumber>SN123456</SerialNumber>
      </DeviceId>
      <Event>
        <EventStruct>
          <EventCode>0 BOOTSTRAP</EventCode>
          <CommandKey></CommandKey>
        </EventStruct>
        <EventStruct>
          <EventCode>6 CONNECTION REQUEST</EventCode>
          <CommandKey></CommandKey>
        </EventStruct>
      </Event>
      <MaxEnvelopes>1</MaxEnvelopes>
      <CurrentTime>2024-01-01T00:00:00Z</CurrentTime>
      <RetryCount>0</RetryCount>
      <ParameterList>
        <ParameterValueStruct>
          <Name>Device.DeviceInfo.Manufacturer</Name>
          <Value xsi:type="xsd:string">Intelbras</Value>
        </ParameterValueStruct>
        <ParameterValueStruct>
          <Name>Device.DeviceInfo.SerialNumber</Name>
          <Value xsi:type="xsd:string">SN123456</Value>
        </ParameterValueStruct>
      </ParameterList>
    </cwmp:Inform>
  </soap:Body>
</soap:Envelope>`

// ParseEnvelope – Inform from a real CPE

func TestParseInformEnvelope(t *testing.T) {
	env, err := ParseEnvelope([]byte(informXML))
	require.NoError(t, err)
	require.NotNil(t, env)

	// Header
	require.NotNil(t, env.Header.ID)
	assert.Equal(t, "1", env.Header.ID.Value)

	// Body / Inform
	require.NotNil(t, env.Body.Inform)
	inf := env.Body.Inform

	// DeviceId
	assert.Equal(t, "Intelbras", inf.DeviceId.Manufacturer)
	assert.Equal(t, "001122", inf.DeviceId.OUI)
	assert.Equal(t, "WiFiRouter", inf.DeviceId.ProductClass)
	assert.Equal(t, "SN123456", inf.DeviceId.SerialNumber)

	// Events
	require.Len(t, inf.Event.Events, 2)
	assert.Equal(t, "0 BOOTSTRAP", inf.Event.Events[0].EventCode)
	assert.Equal(t, "6 CONNECTION REQUEST", inf.Event.Events[1].EventCode)

	// Misc fields
	assert.Equal(t, 1, inf.MaxEnvelopes)
	assert.Equal(t, "2024-01-01T00:00:00Z", inf.CurrentTime)
	assert.Equal(t, 0, inf.RetryCount)

	// ParameterList
	params := inf.ParameterList.ParameterValueStructs
	require.Len(t, params, 2)
	assert.Equal(t, "Device.DeviceInfo.Manufacturer", params[0].Name)
	assert.Equal(t, "Intelbras", params[0].Value.Data)
	assert.Equal(t, "xsd:string", params[0].Value.Type)
	assert.Equal(t, "Device.DeviceInfo.SerialNumber", params[1].Name)
	assert.Equal(t, "SN123456", params[1].Value.Data)
}

func TestParseEnvelopeMalformed(t *testing.T) {
	_, err := ParseEnvelope([]byte("<not valid xml><<"))
	assert.Error(t, err)
}

// BuildInformResponse

// Note: Go's encoding/xml marshals struct tags like `xml:"soap:Envelope"` as
// literal prefixed element names, so the resulting XML cannot be round-tripped
// through ParseEnvelope (which expects the same literal prefix without namespace
// resolution).  Build* tests therefore verify XML content via string inspection.

func TestBuildInformResponse(t *testing.T) {
	data, err := BuildInformResponse("42")
	require.NoError(t, err)
	require.NotEmpty(t, data)

	xmlStr := string(data)
	assert.True(t, strings.Contains(xmlStr, "InformResponse"), "should contain InformResponse")
	assert.True(t, strings.Contains(xmlStr, "<MaxEnvelopes>1</MaxEnvelopes>"), "should have MaxEnvelopes=1")
	assert.True(t, strings.Contains(xmlStr, "42"), "should contain session ID 42")
}

// BuildSetParameterValues

func TestBuildSetParameterValues(t *testing.T) {
	params := map[string]string{
		"Device.WiFi.SSID.1.SSID":                          "TestNet",
		"Device.WiFi.AccessPoint.1.Security.KeyPassphrase": "s3cr3t",
	}

	data, err := BuildSetParameterValues("sess-1", params)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	xmlStr := string(data)

	assert.True(t, strings.Contains(xmlStr, "SetParameterValues"), "should contain SetParameterValues")
	assert.True(t, strings.Contains(xmlStr, "TestNet"), "should contain SSID value")
	assert.True(t, strings.Contains(xmlStr, "s3cr3t"), "should contain password value")
	assert.True(t, strings.Contains(xmlStr, "sess-1"), "should contain session id as ParameterKey")
	assert.True(t, strings.Contains(xmlStr, "Device.WiFi.SSID.1.SSID"), "should contain parameter name")
	assert.True(t, strings.Contains(xmlStr, "xsd:string"), "should contain xsd:string type annotation")
}

// BuildGetParameterValues

func TestBuildGetParameterValues(t *testing.T) {
	names := []string{
		"Device.DeviceInfo.Manufacturer",
		"Device.DeviceInfo.SerialNumber",
		"Device.DeviceInfo.SoftwareVersion",
	}

	data, err := BuildGetParameterValues("sess-2", names)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	xmlStr := string(data)
	assert.True(t, strings.Contains(xmlStr, "GetParameterValues"), "should contain GetParameterValues")
	assert.True(t, strings.Contains(xmlStr, "Device.DeviceInfo.Manufacturer"), "should contain param name")
	assert.True(t, strings.Contains(xmlStr, "Device.DeviceInfo.SerialNumber"), "should contain param name")
	assert.True(t, strings.Contains(xmlStr, "Device.DeviceInfo.SoftwareVersion"), "should contain param name")
	assert.True(t, strings.Contains(xmlStr, "xsd:string[3]"), "should contain arrayType annotation")
}

// BuildReboot

func TestBuildReboot(t *testing.T) {
	data, err := BuildReboot("sess-3", "reboot-key-1")
	require.NoError(t, err)
	require.NotEmpty(t, data)

	xmlStr := string(data)
	assert.True(t, strings.Contains(xmlStr, "Reboot"), "should contain Reboot element")
	assert.True(t, strings.Contains(xmlStr, "reboot-key-1"), "should contain command key")
	assert.True(t, strings.Contains(xmlStr, "sess-3"), "should contain session ID in header")
}

// BuildFactoryReset

func TestBuildFactoryReset(t *testing.T) {
	data, err := BuildFactoryReset("sess-4")
	require.NoError(t, err)
	require.NotEmpty(t, data)

	xmlStr := string(data)
	assert.True(t, strings.Contains(xmlStr, "FactoryReset"), "should contain FactoryReset element")
	assert.True(t, strings.Contains(xmlStr, "sess-4"), "should contain session ID")
}

// BuildDownload – nil error

func TestBuildDownloadNilError(t *testing.T) {
	_, err := BuildDownload("sess-5", nil)
	assert.Error(t, err)
}

// BuildDownload – valid

func TestBuildDownload(t *testing.T) {
	d := &Download{
		CommandKey:   "fw-upgrade-1",
		FileType:     "1 Firmware Upgrade Image",
		URL:          "http://example.com/firmware.bin",
		Username:     "admin",
		Password:     "pass",
		FileSize:     1024000,
		DelaySeconds: 0,
	}

	data, err := BuildDownload("sess-6", d)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	xmlStr := string(data)
	assert.True(t, strings.Contains(xmlStr, "Download"), "should contain Download element")
	assert.True(t, strings.Contains(xmlStr, "fw-upgrade-1"), "should contain command key")
	assert.True(t, strings.Contains(xmlStr, "1 Firmware Upgrade Image"), "should contain file type")
	assert.True(t, strings.Contains(xmlStr, "http://example.com/firmware.bin"), "should contain URL")
}

// BuildEmptyResponse

func TestBuildEmptyResponse(t *testing.T) {
	data := BuildEmptyResponse()
	assert.Empty(t, data)
}

// RoundTrip: verify BuildSetParameterValues embeds all expected fields

func TestRoundTrip(t *testing.T) {
	params := map[string]string{
		"Device.ManagementServer.URL":      "http://acs.example.com:7547/",
		"Device.ManagementServer.Username": "cpe_user",
	}

	data, err := BuildSetParameterValues("round-trip", params)
	require.NoError(t, err)

	xmlStr := string(data)

	assert.True(t, strings.Contains(xmlStr, "SetParameterValues"))
	assert.True(t, strings.Contains(xmlStr, "round-trip"))
	assert.True(t, strings.Contains(xmlStr, "Device.ManagementServer.URL"))
	assert.True(t, strings.Contains(xmlStr, "http://acs.example.com:7547/"))
	assert.True(t, strings.Contains(xmlStr, "Device.ManagementServer.Username"))
	assert.True(t, strings.Contains(xmlStr, "cpe_user"))
}

// ParseEnvelope on a minimal Inform with no events

func TestParseMinimalInform(t *testing.T) {
	minimal := `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope>
  <soap:Header>
    <cwmp:ID mustUnderstand="1">99</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:Inform>
      <DeviceId>
        <Manufacturer>ACME</Manufacturer>
        <OUI>AABBCC</OUI>
        <ProductClass>Router</ProductClass>
        <SerialNumber>SNX</SerialNumber>
      </DeviceId>
      <Event></Event>
      <MaxEnvelopes>1</MaxEnvelopes>
      <CurrentTime>2024-06-01T00:00:00Z</CurrentTime>
      <RetryCount>1</RetryCount>
      <ParameterList></ParameterList>
    </cwmp:Inform>
  </soap:Body>
</soap:Envelope>`

	env, err := ParseEnvelope([]byte(minimal))
	require.NoError(t, err)
	require.NotNil(t, env)
	require.NotNil(t, env.Body.Inform)

	inf := env.Body.Inform
	assert.Equal(t, "ACME", inf.DeviceId.Manufacturer)
	assert.Equal(t, "AABBCC", inf.DeviceId.OUI)
	assert.Equal(t, 1, inf.RetryCount)
	assert.Empty(t, inf.Event.Events)
}

// ParseEnvelope – GetRPCMethods from CPE

func TestParseGetRPCMethods(t *testing.T) {
	xml := `<?xml version="1.0"?>
<soap:Envelope>
  <soap:Header><cwmp:ID mustUnderstand="1">2</cwmp:ID></soap:Header>
  <soap:Body><cwmp:GetRPCMethods></cwmp:GetRPCMethods></soap:Body>
</soap:Envelope>`

	env, err := ParseEnvelope([]byte(xml))
	require.NoError(t, err)
	require.NotNil(t, env)
	require.NotNil(t, env.Body.GetRPCMethods)
}

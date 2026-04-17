package cwmp

import (
	"bytes"
	"encoding/xml"
	"fmt"
)

// XML namespace constants used across all CWMP envelopes.
const (
	soapEnvNS = "http://schemas.xmlsoap.org/soap/envelope/"
	cwmpNS    = "urn:dslforum-org:cwmp-1-2"
	xsdNS     = "http://www.w3.org/2001/XMLSchema"
	xsiNS     = "http://www.w3.org/2001/XMLSchema-instance"
)

// Top-level SOAP envelope

// Envelope is the top-level SOAP 1.1 envelope. It carries CWMP Header and Body
// content. Namespace attributes are written out explicitly so that consumer CPEs
// that do not support default-namespace inheritance can parse the document.
type Envelope struct {
	XMLName xml.Name `xml:"soap:Envelope"`
	NSsoap  string   `xml:"xmlns:soap,attr"`
	NScwmp  string   `xml:"xmlns:cwmp,attr"`
	NSxsd   string   `xml:"xmlns:xsd,attr"`
	NSxsi   string   `xml:"xmlns:xsi,attr"`
	Header  Header   `xml:"soap:Header"`
	Body    Body     `xml:"soap:Body"`
}

// Header carries the optional CWMP session-ID element.
type Header struct {
	ID *HdrID `xml:"cwmp:ID,omitempty"`
}

// HdrID is the cwmp:ID element inside the SOAP header. mustUnderstand is set
// to "1" on ACS-originated messages per TR-069 §3.7.1.
type HdrID struct {
	MustUnderstand string `xml:"soap:mustUnderstand,attr"`
	Value          string `xml:",chardata"`
}

// SOAP Body  union of all possible CWMP messages

// Body holds exactly one CWMP message payload. Pointer fields that are nil are
// omitted from the serialised XML.
type Body struct {
	// CPE → ACS messages
	Inform           *InformRequest    `xml:"cwmp:Inform,omitempty"`
	TransferComplete *TransferComplete `xml:"cwmp:TransferComplete,omitempty"`
	GetRPCMethods    *GetRPCMethods    `xml:"cwmp:GetRPCMethods,omitempty"`

	// ACS → CPE responses / requests
	InformResponse             *InformResponse             `xml:"cwmp:InformResponse,omitempty"`
	GetRPCMethodsResponse      *GetRPCMethodsResponse      `xml:"cwmp:GetRPCMethodsResponse,omitempty"`
	GetParameterValues         *GetParameterValues         `xml:"cwmp:GetParameterValues,omitempty"`
	GetParameterValuesResponse *GetParameterValuesResponse `xml:"cwmp:GetParameterValuesResponse,omitempty"`
	SetParameterValues         *SetParameterValues         `xml:"cwmp:SetParameterValues,omitempty"`
	SetParameterValuesResponse *SetParameterValuesResponse `xml:"cwmp:SetParameterValuesResponse,omitempty"`
	GetParameterNames          *GetParameterNames          `xml:"cwmp:GetParameterNames,omitempty"`
	GetParameterNamesResponse  *GetParameterNamesResponse  `xml:"cwmp:GetParameterNamesResponse,omitempty"`
	AddObject                  *AddObject                  `xml:"cwmp:AddObject,omitempty"`
	AddObjectResponse          *AddObjectResponse          `xml:"cwmp:AddObjectResponse,omitempty"`
	DeleteObject               *DeleteObject               `xml:"cwmp:DeleteObject,omitempty"`
	DeleteObjectResponse       *DeleteObjectResponse       `xml:"cwmp:DeleteObjectResponse,omitempty"`
	Download                   *Download                   `xml:"cwmp:Download,omitempty"`
	DownloadResponse           *DownloadResponse           `xml:"cwmp:DownloadResponse,omitempty"`
	Reboot                     *Reboot                     `xml:"cwmp:Reboot,omitempty"`
	RebootResponse             *RebootResponse             `xml:"cwmp:RebootResponse,omitempty"`
	FactoryReset               *FactoryReset               `xml:"cwmp:FactoryReset,omitempty"`
	FactoryResetResponse       *FactoryResetResponse       `xml:"cwmp:FactoryResetResponse,omitempty"`
	Fault                      *SOAPFault                  `xml:"soap:Fault,omitempty"`
}

// Inform (CPE -> ACS)

// InformRequest is sent by the CPE at session open and on triggered events.
type InformRequest struct {
	DeviceId      DeviceID           `xml:"DeviceId"`
	Event         EventList          `xml:"Event"`
	MaxEnvelopes  int                `xml:"MaxEnvelopes"`
	CurrentTime   string             `xml:"CurrentTime"`
	RetryCount    int                `xml:"RetryCount"`
	ParameterList ParameterValueList `xml:"ParameterList"`
}

// DeviceID contains the four mandatory CPE identity fields.
type DeviceID struct {
	Manufacturer string `xml:"Manufacturer"`
	OUI          string `xml:"OUI"`
	ProductClass string `xml:"ProductClass"`
	SerialNumber string `xml:"SerialNumber"`
}

// EventList wraps the slice of EventStruct elements.
type EventList struct {
	Events []EventStruct `xml:"EventStruct"`
}

// EventStruct describes a single TR-069 event (e.g. "0 BOOTSTRAP").
type EventStruct struct {
	EventCode  string `xml:"EventCode"`
	CommandKey string `xml:"CommandKey"`
}

// ParameterValueList wraps a slice of ParameterValueStruct items. It is reused
// in Inform, GetParameterValuesResponse, and SetParameterValues.
type ParameterValueList struct {
	ParameterValueStructs []ParameterValueStruct `xml:"ParameterValueStruct"`
}

// ParameterValueStruct holds one name/value pair with the xsi:type annotation
// carried on the Value element.
type ParameterValueStruct struct {
	Name  string `xml:"Name"`
	Value Value  `xml:"Value"`
}

// Value holds the typed string content of a CWMP parameter value.
// The type attribute is written without a namespace prefix so that it is
// round-trippable through the normalizeXML preprocessor used by ParseEnvelope.
type Value struct {
	Type string `xml:"type,attr,omitempty"`
	Data string `xml:",chardata"`
}

// InformResponse is the ACS reply to an Inform message.
type InformResponse struct {
	MaxEnvelopes int `xml:"MaxEnvelopes"`
}

// GetRPCMethods (CPE → ACS / ACS → CPE)

// GetRPCMethods is an empty request body – the XML element itself is the signal.
type GetRPCMethods struct{}

// GetRPCMethodsResponse lists the RPC methods supported by the responder.
type GetRPCMethodsResponse struct {
	MethodList MethodList `xml:"MethodList"`
}

// MethodList holds the list of method name strings.
type MethodList struct {
	Methods []string `xml:"string"`
}

// GetParameterValues (ACS → CPE)

// GetParameterValues requests current values for the listed parameter paths.
type GetParameterValues struct {
	ParameterNames ParameterNames `xml:"ParameterNames"`
}

// ParameterNames carries the arrayType SOAP-ENC attribute and the list of paths.
type ParameterNames struct {
	ArrayType string   `xml:"soap-enc:arrayType,attr,omitempty"`
	Names     []string `xml:"string"`
}

// GetParameterValuesResponse carries the requested parameter values.
type GetParameterValuesResponse struct {
	ParameterList ParameterValueList `xml:"ParameterList"`
}

// SetParameterValues (ACS → CPE)

// SetParameterValues requests that the CPE update the given parameters.
type SetParameterValues struct {
	ParameterList ParameterValueList `xml:"ParameterList"`
	ParameterKey  string             `xml:"ParameterKey"`
}

// SetParameterValuesResponse reports whether the CPE requires a reboot.
// Status 0 = no reboot needed, Status 1 = reboot required.
type SetParameterValuesResponse struct {
	Status int `xml:"Status"`
}

// GetParameterNames (ACS → CPE)

// GetParameterNames requests the parameter tree under ParameterPath.
type GetParameterNames struct {
	ParameterPath string `xml:"ParameterPath"`
	NextLevel     bool   `xml:"NextLevel"`
}

// GetParameterNamesResponse carries the discovered parameter metadata.
type GetParameterNamesResponse struct {
	ParameterList ParameterInfoList `xml:"ParameterList"`
}

// ParameterInfoList wraps a slice of ParameterInfoStruct items.
type ParameterInfoList struct {
	ParameterInfoStructs []ParameterInfoStruct `xml:"ParameterInfoStruct"`
}

// ParameterInfoStruct describes a single parameter's path and write-ability.
type ParameterInfoStruct struct {
	Name     string `xml:"Name"`
	Writable bool   `xml:"Writable"`
}

// AddObject / DeleteObject (ACS → CPE)

// AddObject requests the CPE to create a new instance of a multi-instance object.
type AddObject struct {
	ObjectName   string `xml:"ObjectName"`
	ParameterKey string `xml:"ParameterKey"`
}

// AddObjectResponse returns the new instance number and reboot status.
type AddObjectResponse struct {
	InstanceNumber int `xml:"InstanceNumber"`
	Status         int `xml:"Status"`
}

// DeleteObject requests the CPE to remove an existing object instance.
type DeleteObject struct {
	ObjectName   string `xml:"ObjectName"`
	ParameterKey string `xml:"ParameterKey"`
}

// DeleteObjectResponse reports the reboot status.
type DeleteObjectResponse struct {
	Status int `xml:"Status"`
}

// Download (ACS → CPE)

// Download instructs the CPE to download a file from the given URL.
type Download struct {
	CommandKey     string `xml:"CommandKey"`
	FileType       string `xml:"FileType"`
	URL            string `xml:"URL"`
	Username       string `xml:"Username"`
	Password       string `xml:"Password"`
	FileSize       int    `xml:"FileSize"`
	TargetFileName string `xml:"TargetFileName"`
	DelaySeconds   int    `xml:"DelaySeconds"`
}

// DownloadResponse carries the initial status of the download operation.
// Status 0 = download completed, Status 1 = still in progress.
type DownloadResponse struct {
	Status       int    `xml:"Status"`
	StartTime    string `xml:"StartTime"`
	CompleteTime string `xml:"CompleteTime"`
}

// TransferComplete (CPE → ACS)

// TransferComplete notifies the ACS that a previously requested transfer has
// finished (successfully or with an error).
type TransferComplete struct {
	CommandKey   string      `xml:"CommandKey"`
	FaultStruct  FaultStruct `xml:"FaultStruct"`
	StartTime    string      `xml:"StartTime"`
	CompleteTime string      `xml:"CompleteTime"`
}

// FaultStruct is embedded in TransferComplete to convey success (code 0) or a
// CWMP fault code.
type FaultStruct struct {
	FaultCode   int    `xml:"FaultCode"`
	FaultString string `xml:"FaultString"`
}

// Reboot / FactoryReset (ACS → CPE)

// Reboot instructs the CPE to reboot.
type Reboot struct {
	CommandKey string `xml:"CommandKey"`
}

// RebootResponse is the empty acknowledgement of a Reboot request.
type RebootResponse struct{}

// FactoryReset instructs the CPE to restore factory defaults.
type FactoryReset struct{}

// FactoryResetResponse is the empty acknowledgement of a FactoryReset request.
type FactoryResetResponse struct{}

// SOAP Fault

// SOAPFault wraps a CWMP fault inside a standard SOAP 1.1 Fault element.
type SOAPFault struct {
	FaultCode   string      `xml:"faultcode"`
	FaultString string      `xml:"faultstring"`
	Detail      FaultDetail `xml:"detail"`
}

// FaultDetail wraps the CWMP-specific fault information.
type FaultDetail struct {
	CWMPFault CWMPFault `xml:"cwmp:Fault"`
}

// CWMPFault carries the TR-069 fault code and human-readable description.
type CWMPFault struct {
	FaultCode   string `xml:"FaultCode"`
	FaultString string `xml:"FaultString"`
}

// Build helpers

// BuildEnvelope marshals body into a complete SOAP envelope with the given
// session ID. The returned bytes include the XML declaration header.
func BuildEnvelope(id string, body Body) ([]byte, error) {
	env := Envelope{
		NSsoap: soapEnvNS,
		NScwmp: cwmpNS,
		NSxsd:  xsdNS,
		NSxsi:  xsiNS,
		Header: Header{
			ID: &HdrID{
				MustUnderstand: "1",
				Value:          id,
			},
		},
		Body: body,
	}

	var buf bytes.Buffer
	_, _ = buf.WriteString(xml.Header)

	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(env); err != nil {
		return nil, fmt.Errorf("cwmp: failed to encode SOAP envelope: %w", err)
	}
	if err := enc.Flush(); err != nil {
		return nil, fmt.Errorf("cwmp: failed to flush SOAP encoder: %w", err)
	}
	return buf.Bytes(), nil
}

// BuildInformResponse constructs a SOAP envelope containing an InformResponse
// with MaxEnvelopes set to 1.
func BuildInformResponse(id string) ([]byte, error) {
	body := Body{
		InformResponse: &InformResponse{
			MaxEnvelopes: 1,
		},
	}
	return BuildEnvelope(id, body)
}

// BuildGetParameterValues constructs a GetParameterValues request envelope
// targeting the given parameter path names.
func BuildGetParameterValues(id string, names []string) ([]byte, error) {
	body := Body{
		GetParameterValues: &GetParameterValues{
			ParameterNames: ParameterNames{
				ArrayType: fmt.Sprintf("xsd:string[%d]", len(names)),
				Names:     names,
			},
		},
	}
	return BuildEnvelope(id, body)
}

// BuildSetParameterValues constructs a SetParameterValues request envelope
// from a flat map of parameter path → string value. All values are typed as
// xsd:string; callers that need specific xsd types should build the
// ParameterValueList directly and call BuildEnvelope.
func BuildSetParameterValues(id string, params map[string]string) ([]byte, error) {
	pvs := make([]ParameterValueStruct, 0, len(params))
	for name, val := range params {
		pvs = append(pvs, ParameterValueStruct{
			Name: name,
			Value: Value{
				Type: "xsd:string",
				Data: val,
			},
		})
	}
	body := Body{
		SetParameterValues: &SetParameterValues{
			ParameterList: ParameterValueList{
				ParameterValueStructs: pvs,
			},
			ParameterKey: id,
		},
	}
	return BuildEnvelope(id, body)
}

// BuildReboot constructs a Reboot request envelope.
func BuildReboot(id string, commandKey string) ([]byte, error) {
	body := Body{
		Reboot: &Reboot{
			CommandKey: commandKey,
		},
	}
	return BuildEnvelope(id, body)
}

// BuildFactoryReset constructs a FactoryReset request envelope.
func BuildFactoryReset(id string) ([]byte, error) {
	body := Body{
		FactoryReset: &FactoryReset{},
	}
	return BuildEnvelope(id, body)
}

// BuildDownload constructs a Download request envelope from a populated
// Download struct. The caller is responsible for filling in all required fields.
func BuildDownload(id string, d *Download) ([]byte, error) {
	if d == nil {
		return nil, fmt.Errorf("cwmp: BuildDownload called with nil Download")
	}
	body := Body{
		Download: d,
	}
	return BuildEnvelope(id, body)
}

// BuildAddObject constructs an AddObject request envelope asking the CPE to
// create a new instance of the given multi-instance object.
func BuildAddObject(id, objectName string) ([]byte, error) {
	body := Body{
		AddObject: &AddObject{
			ObjectName:   objectName,
			ParameterKey: id,
		},
	}
	return BuildEnvelope(id, body)
}

// BuildDeleteObject constructs a DeleteObject request envelope asking the CPE
// to remove the given object instance.
func BuildDeleteObject(id, objectName string) ([]byte, error) {
	body := Body{
		DeleteObject: &DeleteObject{
			ObjectName:   objectName,
			ParameterKey: id,
		},
	}
	return BuildEnvelope(id, body)
}

// BuildEmptyResponse returns a minimal empty HTTP body that signals to the CPE
// that the ACS has no further requests to issue for this session. Per TR-069
// the ACS returns an HTTP 200 with an empty body (zero bytes) to end
// the session.
func BuildEmptyResponse() []byte {
	return []byte{}
}

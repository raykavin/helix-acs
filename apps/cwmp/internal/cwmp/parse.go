package cwmp

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
)

// Incoming SOAP parse types
// Real CPE devices use a wide variety of namespace prefix styles, so we
// normalise the raw XML before decoding: all namespace prefixes are stripped
// from element and attribute names, and xmlns declarations are removed.
// These "in*" types therefore use plain (unprefixed) XML tags.

type inEnvelope struct {
	XMLName xml.Name `xml:"Envelope"`
	Header  inHeader `xml:"Header"`
	Body    inBody   `xml:"Body"`
}

type inHeader struct {
	ID *inHdrID `xml:"ID,omitempty"`
}

type inHdrID struct {
	MustUnderstand string `xml:"mustUnderstand,attr"`
	Value          string `xml:",chardata"`
}

type inBody struct {
	Inform                     *InformRequest              `xml:"Inform,omitempty"`
	TransferComplete           *TransferComplete           `xml:"TransferComplete,omitempty"`
	GetRPCMethods              *GetRPCMethods              `xml:"GetRPCMethods,omitempty"`
	InformResponse             *InformResponse             `xml:"InformResponse,omitempty"`
	GetRPCMethodsResponse      *GetRPCMethodsResponse      `xml:"GetRPCMethodsResponse,omitempty"`
	GetParameterValues         *GetParameterValues         `xml:"GetParameterValues,omitempty"`
	GetParameterValuesResponse *GetParameterValuesResponse `xml:"GetParameterValuesResponse,omitempty"`
	SetParameterValues         *SetParameterValues         `xml:"SetParameterValues,omitempty"`
	SetParameterValuesResponse *SetParameterValuesResponse `xml:"SetParameterValuesResponse,omitempty"`
	GetParameterNames          *GetParameterNames          `xml:"GetParameterNames,omitempty"`
	GetParameterNamesResponse  *GetParameterNamesResponse  `xml:"GetParameterNamesResponse,omitempty"`
	AddObject                  *AddObject                  `xml:"AddObject,omitempty"`
	AddObjectResponse          *AddObjectResponse          `xml:"AddObjectResponse,omitempty"`
	DeleteObject               *DeleteObject               `xml:"DeleteObject,omitempty"`
	DeleteObjectResponse       *DeleteObjectResponse       `xml:"DeleteObjectResponse,omitempty"`
	Download                   *Download                   `xml:"Download,omitempty"`
	DownloadResponse           *DownloadResponse           `xml:"DownloadResponse,omitempty"`
	Reboot                     *Reboot                     `xml:"Reboot,omitempty"`
	RebootResponse             *RebootResponse             `xml:"RebootResponse,omitempty"`
	FactoryReset               *FactoryReset               `xml:"FactoryReset,omitempty"`
	FactoryResetResponse       *FactoryResetResponse       `xml:"FactoryResetResponse,omitempty"`
	Fault                      *SOAPFault                  `xml:"Fault,omitempty"`
}

// ParseEnvelope

// ParseEnvelope normalises incoming SOAP/XML bytes (stripping all namespace
// prefixes) and unmarshals them into an Envelope. It handles the full range of
// namespace declaration styles emitted by real CPE devices.
func ParseEnvelope(data []byte) (*Envelope, error) {
	normalized := normalizeXML(data)

	var in inEnvelope
	if err := xml.Unmarshal(normalized, &in); err != nil {
		return nil, fmt.Errorf("cwmp: failed to parse SOAP envelope: %w", err)
	}

	env := &Envelope{
		Body: Body{
			Inform:                     in.Body.Inform,
			TransferComplete:           in.Body.TransferComplete,
			GetRPCMethods:              in.Body.GetRPCMethods,
			InformResponse:             in.Body.InformResponse,
			GetRPCMethodsResponse:      in.Body.GetRPCMethodsResponse,
			GetParameterValues:         in.Body.GetParameterValues,
			GetParameterValuesResponse: in.Body.GetParameterValuesResponse,
			SetParameterValues:         in.Body.SetParameterValues,
			SetParameterValuesResponse: in.Body.SetParameterValuesResponse,
			GetParameterNames:          in.Body.GetParameterNames,
			GetParameterNamesResponse:  in.Body.GetParameterNamesResponse,
			AddObject:                  in.Body.AddObject,
			AddObjectResponse:          in.Body.AddObjectResponse,
			DeleteObject:               in.Body.DeleteObject,
			DeleteObjectResponse:       in.Body.DeleteObjectResponse,
			Download:                   in.Body.Download,
			DownloadResponse:           in.Body.DownloadResponse,
			Reboot:                     in.Body.Reboot,
			RebootResponse:             in.Body.RebootResponse,
			FactoryReset:               in.Body.FactoryReset,
			FactoryResetResponse:       in.Body.FactoryResetResponse,
			Fault:                      in.Body.Fault,
		},
	}
	if in.Header.ID != nil {
		env.Header = Header{
			ID: &HdrID{
				MustUnderstand: in.Header.ID.MustUnderstand,
				Value:          in.Header.ID.Value,
			},
		}
	}
	return env, nil
}

// XML normalizer

// normalizeXML strips all namespace prefixes from element and attribute names
// and removes xmlns declarations. The resulting XML uses only local names,
// making it decodable with simple (unprefixed) struct tags regardless of
// which namespace prefix style the sending CPE uses.
func normalizeXML(data []byte) []byte {
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false

	var buf bytes.Buffer
	enc := xml.NewEncoder(&buf)

	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			t.Name = xml.Name{Local: localPart(t.Name.Local)}
			var attrs []xml.Attr
			for _, a := range t.Attr {
				// Drop namespace declarations (xmlns:prefix="..." and xmlns="...")
				if a.Name.Space == "xmlns" {
					continue
				}
				if a.Name.Space == "" && (a.Name.Local == "xmlns" || strings.HasPrefix(a.Name.Local, "xmlns:")) {
					continue
				}
				// Strip namespace prefix from attribute name
				a.Name = xml.Name{Local: localPart(a.Name.Local)}
				attrs = append(attrs, a)
			}
			t.Attr = attrs
			_ = enc.EncodeToken(t)
		case xml.EndElement:
			t.Name = xml.Name{Local: localPart(t.Name.Local)}
			_ = enc.EncodeToken(t)
		default:
			_ = enc.EncodeToken(tok)
		}
	}
	_ = enc.Flush()
	return buf.Bytes()
}

// localPart returns the local (post-colon) part of a potentially prefixed name.
// "soap:Envelope" → "Envelope", "Envelope" → "Envelope".
func localPart(s string) string {
	if i := strings.LastIndex(s, ":"); i >= 0 {
		return s[i+1:]
	}
	return s
}

// Package schema provides a filesystem-driven abstraction layer for TR-069
// parameter path definitions. Each YAML file describes one operation domain
// (wifi, wan, change_password, …) for a specific data model (tr181 / tr098)
// and an optional vendor override. The SchemaMapper implements the
// datamodel.Mapper interface so the rest of the CWMP stack needs no changes.
package schema

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Parameter describes a single TR-069 parameter: its semantic name, the
// raw path template (possibly containing {placeholder} tokens), and its
// xsd type used in SOAP SetParameterValues requests.
type Parameter struct {
	Name string `yaml:"name"`
	Path string `yaml:"path"`
	Type string `yaml:"type"`
}

// Schema represents one YAML schema file.  A file covers one operation domain
// (e.g. "wifi", "wan", "change_password") for one model and optional vendor.
type Schema struct {
	ID          string      `yaml:"id"`
	Model       string      `yaml:"model"`
	Vendor      string      `yaml:"vendor,omitempty"`
	Description string      `yaml:"description,omitempty"`
	Parameters  []Parameter `yaml:"parameters"`
}

// LoadFile reads and unmarshals a single YAML schema file from disk.
func LoadFile(path string) (*Schema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read schema file %q: %w", path, err)
	}
	var s Schema
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse schema file %q: %w", path, err)
	}
	return &s, nil
}

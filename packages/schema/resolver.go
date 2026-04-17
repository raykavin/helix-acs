package schema

import (
	"strings"
)

// Resolver selects the correct schema name for a device based on its
// manufacturer, product class, and data model.
//
// Resolution priority (highest first):
//  1. vendor/<normalised_manufacturer>/<data_model>  – vendor-specific schema
//  2. <data_model>                                   – generic model schema
type Resolver struct {
	registry *Registry
}

// NewResolver returns a Resolver backed by the given Registry.
func NewResolver(reg *Registry) *Resolver {
	return &Resolver{registry: reg}
}

// Resolve returns the schema name to use for a device.
//
//   - manufacturer: reported by the CPE (e.g. "Huawei Technologies Co., Ltd.")
//   - productClass: reported by the CPE (e.g. "HG8245H")
//   - dataModel:    "tr181" or "tr098" (detected from Inform parameter prefixes)
//
// The returned value is stored on the Device record and used to look up schemas
// for every subsequent provisioning operation.  Examples:
//
//	"tr181"                  – generic TR-181
//	"vendor/huawei/tr181"    – Huawei TR-181 override
//	"vendor/zte/tr098"       – ZTE TR-098 override
func (r *Resolver) Resolve(manufacturer, _ /* productClass */, dataModel string) string {
	vendor := normaliseVendor(manufacturer)

	if vendor != "" {
		candidate := "vendor/" + vendor + "/" + dataModel
		if r.registry.Has(candidate) {
			return candidate
		}
	}

	// Fall back to the generic model schema.
	if dataModel == "" {
		dataModel = "tr098"
	}
	return dataModel
}

// normaliseVendor converts a free-form manufacturer string into a short,
// lower-case, filesystem-safe vendor slug.
//
// Examples:
//
//	"Huawei Technologies Co., Ltd."  → "huawei"
//	"ZTE Corporation"                → "zte"
//	"Intelbras S/A"                  → "intelbras"
//	""                               → ""
func normaliseVendor(manufacturer string) string {
	s := strings.ToLower(strings.TrimSpace(manufacturer))
	if s == "" {
		return ""
	}

	// Well-known vendor prefixes → canonical slug.
	for prefix, slug := range knownVendors {
		if strings.HasPrefix(s, prefix) {
			return slug
		}
	}

	// Generic fallback: take the first word and strip non-alphanumeric chars.
	word := strings.FieldsFunc(s, func(r rune) bool {
		return r == ' ' || r == ',' || r == '.' || r == '/' || r == '\\'
	})[0]

	return word
}

// knownVendors maps lower-case manufacturer name prefixes to canonical slugs.
// Add new entries here or extend with a config file as needed.
var knownVendors = map[string]string{
	"huawei":   "huawei",
	"zte":      "zte",
	"tp-link":  "tplink",
	"tp link":  "tplink",
	"tplink":   "tplink",
	"fiberhome": "fiberhome",
	"intelbras": "intelbras",
	"datacom":  "datacom",
	"nokia":    "nokia",
	"ericsson": "ericsson",
	"sagemcom": "sagemcom",
	"technicolor": "technicolor",
	"arcadyan": "arcadyan",
	"sercomm":  "sercomm",
	"askey":    "askey",
}

package datamodel

import (
	"fmt"
	"strconv"
	"time"
)

// iso8601Formats lists the datetime layouts tried in order when validating a
// TypeDateTime value. TR-069 mandates ISO 8601 / RFC 3339 formatting.
var iso8601Formats = []string{
	time.RFC3339,           // 2006-01-02T15:04:05Z07:00
	time.RFC3339Nano,       // 2006-01-02T15:04:05.999999999Z07:00
	"2006-01-02T15:04:05",  // no timezone suffix
	"2006-01-02T15:04:05Z", // explicit UTC Z
	"2006-01-02",           // date only
}

// ValidateType validates value against the given xsd type string. It returns a
// descriptive error when the value cannot be parsed as the stated type, or nil
// on success. TypeString always passes. An unknown xsdType is silently accepted
// so that future or vendor-extended types do not break processing.
func ValidateType(xsdType, value string) error {
	switch xsdType {
	case TypeBoolean:
		return validateBoolean(value)
	case TypeUnsignedInt:
		return validateUnsignedInt(value)
	case TypeInt:
		return validateInt(value)
	case TypeDateTime:
		return validateDateTime(value)
	case TypeString, TypeBase64:
		// No structural validation required for these types.
		return nil
	default:
		// Unknown / vendor-extended type: accept.
		return nil
	}
}

// validateBoolean accepts "0", "1", "true", and "false" (case-sensitive per
// TR-069 §A.2.3).
func validateBoolean(value string) error {
	switch value {
	case "0", "1", "true", "false":
		return nil
	}
	return fmt.Errorf("datamodel: invalid boolean value %q: must be one of 0, 1, true, false", value)
}

// validateUnsignedInt requires the value to be a valid non-negative integer
// that fits in a uint64.
func validateUnsignedInt(value string) error {
	if _, err := strconv.ParseUint(value, 10, 64); err != nil {
		return fmt.Errorf("datamodel: invalid unsignedInt value %q: %w", value, err)
	}
	return nil
}

// validateInt requires the value to be a valid signed integer that fits in
// an int64.
func validateInt(value string) error {
	if _, err := strconv.ParseInt(value, 10, 64); err != nil {
		return fmt.Errorf("datamodel: invalid int value %q: %w", value, err)
	}
	return nil
}

// validateDateTime tries each of the known ISO 8601 layouts. The first
// successful parse wins.
func validateDateTime(value string) error {
	// TR-069 uses "0001-01-01T00:00:00Z" as the "unknown time" sentinel;
	// accept it explicitly to avoid layout mismatches on some platforms.
	if value == "0001-01-01T00:00:00Z" || value == "0001-01-01T00:00:00" {
		return nil
	}
	for _, layout := range iso8601Formats {
		if _, err := time.Parse(layout, value); err == nil {
			return nil
		}
	}
	return fmt.Errorf("datamodel: invalid dateTime value %q: does not match any supported ISO 8601 format", value)
}

package schema

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

// Registry holds all loaded schemas indexed by their resolved schema name.
//
// Schema names follow these conventions:
//   - Generic model: "tr181" or "tr098"
//   - Vendor-specific: "vendor/huawei/tr181" or "vendor/zte/tr098"
//
// Each schema name maps to one or more Schema objects (one per YAML file
// loaded for that name).
type Registry struct {
	schemas map[string][]*Schema
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{schemas: make(map[string][]*Schema)}
}

// LoadDir walks the given directory tree and loads every *.yaml file as a
// Schema. The directory structure determines the schema name:
//
//	<root>/tr181/wifi.yaml           → schema name "tr181"
//	<root>/tr098/wan.yaml            → schema name "tr098"
//	<root>/vendors/huawei/tr181/…   → schema name "vendor/huawei/tr181"
//	<root>/vendors/zte/tr098/…      → schema name "vendor/zte/tr098"
func (r *Registry) LoadDir(root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk schema dir: %w", err)
		}
		if d.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return nil
		}

		s, lerr := LoadFile(path)
		if lerr != nil {
			return lerr
		}

		name := schemaNameFromPath(root, path)
		r.schemas[name] = append(r.schemas[name], s)
		return nil
	})
}

// schemaNameFromPath derives the schema name from a file path.
//
// Examples (root = "./schemas"):
//
//	./schemas/tr181/wifi.yaml                     → "tr181"
//	./schemas/tr098/wan.yaml                      → "tr098"
//	./schemas/vendors/huawei/tr181/wifi.yaml      → "vendor/huawei/tr181"
func schemaNameFromPath(root, path string) string {
	// Normalise separators and strip the root prefix.
	root = filepath.ToSlash(filepath.Clean(root))
	path = filepath.ToSlash(filepath.Clean(path))
	rel := strings.TrimPrefix(path, root+"/")

	// Strip the filename, keep the directory portion.
	dir := filepath.ToSlash(filepath.Dir(rel))

	// "vendors/huawei/tr181" → "vendor/huawei/tr181"
	parts := strings.SplitN(dir, "/", 2)
	if parts[0] == "vendors" && len(parts) == 2 {
		return "vendor/" + parts[1]
	}
	return dir // e.g. "tr181" or "tr098"
}

// GetAll returns every Schema loaded under the given schema name.
// Returns nil if nothing is registered.
func (r *Registry) GetAll(schemaName string) []*Schema {
	return r.schemas[schemaName]
}

// Has returns true when at least one schema is registered under schemaName.
func (r *Registry) Has(schemaName string) bool {
	return len(r.schemas[schemaName]) > 0
}

// ParamMap builds a flat parameter-name → path-template map from all schemas
// registered under schemaName.  When the same parameter name appears in
// multiple files, the last one wins.
func (r *Registry) ParamMap(schemaName string) map[string]string {
	out := make(map[string]string)
	for _, s := range r.schemas[schemaName] {
		for _, p := range s.Parameters {
			out[p.Name] = p.Path
		}
	}
	return out
}

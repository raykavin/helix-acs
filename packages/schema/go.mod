module github.com/raykavin/helix-acs/packages/schema

go 1.25.0

require (
	github.com/raykavin/helix-acs/packages/datamodel v0.0.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/kr/text v0.2.0 // indirect
	github.com/rogpeppe/go-internal v1.9.0 // indirect
)

replace github.com/raykavin/helix-acs/packages/datamodel => ../datamodel

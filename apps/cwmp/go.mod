module github.com/raykavin/helix-acs/apps/cwmp

go 1.25.0

require (
	github.com/google/uuid v1.6.0
	github.com/raykavin/gokit v0.0.6
	github.com/raykavin/helix-acs/packages/auth v0.0.0
	github.com/raykavin/helix-acs/packages/config v0.0.0
	github.com/raykavin/helix-acs/packages/datamodel v0.0.0
	github.com/raykavin/helix-acs/packages/device v0.0.0
	github.com/raykavin/helix-acs/packages/logger v0.0.0
	github.com/raykavin/helix-acs/packages/schema v0.0.0
	github.com/raykavin/helix-acs/packages/task v0.0.0
	github.com/redis/go-redis/v9 v9.18.0
	github.com/stretchr/testify v1.11.1
	go.mongodb.org/mongo-driver v1.17.3
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/common-nighthawk/go-figure v0.0.0-20210622060536-734e95fb86be // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/fatih/color v1.19.0 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.2 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/goterm v0.0.0-20200907032337-555d40f16ae2 // indirect
	github.com/klauspost/compress v1.18.5 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/montanaflynn/stats v0.7.1 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/rs/zerolog v1.35.0 // indirect
	github.com/sagikazarmark/locafero v0.11.0 // indirect
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/spf13/viper v1.21.0 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/tj/go-spin v1.1.0 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.48.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/raykavin/helix-acs/packages/auth => ../../packages/auth
	github.com/raykavin/helix-acs/packages/config => ../../packages/config
	github.com/raykavin/helix-acs/packages/datamodel => ../../packages/datamodel
	github.com/raykavin/helix-acs/packages/device => ../../packages/device
	github.com/raykavin/helix-acs/packages/logger => ../../packages/logger
	github.com/raykavin/helix-acs/packages/schema => ../../packages/schema
	github.com/raykavin/helix-acs/packages/task => ../../packages/task
)

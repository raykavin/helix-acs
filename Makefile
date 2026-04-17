IMAGE_NAME ?= helix
VERSION    ?= 1.0.0

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  dev-api         Run API server (go run ./cmd/... inside apps/api)"
	@echo "  dev-cwmp        Run CWMP server (go run ./cmd/... inside apps/cwmp)"
	@echo "  dev             Run both servers concurrently"
	@echo "  build           Build both binaries into bin/"
	@echo "  build-api       Build API binary only"
	@echo "  build-cwmp      Build CWMP binary only"
	@echo "  test            Sync workspace and run all tests"
	@echo "  vet             Run go vet across all workspace modules"
	@echo "  docker-api      Build Docker image for the API server"
	@echo "  docker-cwmp     Build Docker image for the CWMP server"
	@echo "  mocks           Run generate-mocks.sh script"
	@echo "  swagger         Run swagger-doc.sh script"

.PHONY: dev-api
dev-api:
	cd apps/api && go run ./cmd/...

.PHONY: dev-cwmp
dev-cwmp:
	cd apps/cwmp && go run ./cmd/...

.PHONY: dev
dev:
	$(MAKE) -j2 dev-api dev-cwmp

.PHONY: build
build: build-api build-cwmp

.PHONY: build-api
build-api:
	mkdir -p bin
	cd apps/api && go build -o ../../bin/api ./cmd/...

.PHONY: build-cwmp
build-cwmp:
	mkdir -p bin
	cd apps/cwmp && go build -o ../../bin/cwmp ./cmd/...

.PHONY: test
test:
	go work sync && go test ./...

.PHONY: vet
vet:
	go vet ./apps/api/... ./apps/cwmp/... \
	       ./packages/config/... ./packages/logger/... ./packages/auth/... \
	       ./packages/device/... ./packages/datamodel/... ./packages/task/... \
	       ./packages/schema/...

.PHONY: docker-api
docker-api:
	docker build -f apps/api/Dockerfile -t $(IMAGE_NAME)-api:$(VERSION) .

.PHONY: docker-cwmp
docker-cwmp:
	docker build -f apps/cwmp/Dockerfile -t $(IMAGE_NAME)-cwmp:$(VERSION) .

.PHONY: mocks
mocks:
	@scripts/sh/generate-mocks.sh $(MOCK_ARGS)

.PHONY: swagger
swagger:
	@scripts/sh/swagger.sh

IMAGE_NAME ?= ${PROJECT_NAME}
VERSION    ?= 1.0.0

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  dev-api         Run API server locally (go run)"
	@echo "  dev-cwmp        Run CWMP server locally (go run)"
	@echo "  dev             Run both servers concurrently"
	@echo "  build           Compile both binaries to bin/"
	@echo "  test            Run all tests"
	@echo "  docker-api      Build API Docker image (acs-api)"
	@echo "  docker-cwmp     Build CWMP Docker image (acs-cwmp)"
	@echo "  deploy          Run deploy.sh script (uses IMAGE_NAME and VERSION)"
	@echo "  mocks           Run generate-mocks.sh script (pass args as MOCK_ARGS)"
	@echo "  swagger         Run swagger-doc.sh script"

.PHONY: dev-api
dev-api:
	go run ./cmd/api

.PHONY: dev-cwmp
dev-cwmp:
	go run ./cmd/cwmp

.PHONY: dev
dev:
	make -j2 dev-api dev-cwmp

.PHONY: build
build:
	go build -o bin/api  ./cmd/api
	go build -o bin/cwmp ./cmd/cwmp

.PHONY: test
test:
	go test ./...

.PHONY: docker-api
docker-api:
	docker build -f Dockerfile.api -t acs-api .

.PHONY: docker-cwmp
docker-cwmp:
	docker build -f Dockerfile.cwmp -t acs-cwmp .

.PHONY: deploy
deploy:
	@scripts/sh/deploy.sh $(IMAGE_NAME) $(VERSION)

.PHONY: mocks
mocks:
	@scripts/sh/generate-mocks.sh $(MOCK_ARGS)

.PHONY: swagger
swagger:
	@scripts/sh/swagger.sh

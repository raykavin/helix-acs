IMAGE_NAME ?= ${PROJECT_NAME}
VERSION ?= 1.0.0

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build           Run build.sh script (uses IMAGE_NAME and VERSION)"
	@echo "  deploy          Run deploy.sh script (uses IMAGE_NAME and VERSION)"
	@echo "  mocks           Run generate-mocks.sh script (pass args as MOCK_ARGS)"
	@echo "  test            Run test.sh script"
	@echo "  setup           Run setup.sh script"
	@echo "  swagger         Run swagger-doc.sh script"

.PHONY: build
build:
	@scripts/sh/build.sh $(IMAGE_NAME) $(VERSION)

.PHONY: deploy
deploy:
	@scripts/sh/deploy.sh $(IMAGE_NAME) $(VERSION)

.PHONY: mocks
mocks:
	@scripts/sh/generate-mocks.sh $(MOCK_ARGS)

.PHONY: test
test:
	@scripts/sh/test.sh

.PHONY: swagger
swagger:
	@scripts/sh/swagger.sh

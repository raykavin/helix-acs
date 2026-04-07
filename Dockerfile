# Global build arguments
ARG GO_BUILD_BASE_IMAGE=1.25.7-alpine3.23
ARG GO_BUILD_FLAGS=""
ARG PROJECT_NAME="helix"
ARG TZ=America/Belem

# Build stage
FROM golang:${GO_BUILD_BASE_IMAGE} AS build

# Re-declare args for this stage
ARG PROJECT_NAME
ARG GO_BUILD_FLAGS
ARG TZ

ENV APP_NAME=${PROJECT_NAME}
ENV TZ=${TZ}

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev tzdata

# Set timezone
RUN ln -snf /usr/share/zoneinfo/${TZ} /etc/localtime && \
    echo ${TZ} > /etc/timezone

# Copy go mod first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy rest of source
COPY . .

# Build application
RUN CGO_ENABLED=1 go build ${GO_BUILD_FLAGS} -o ${APP_NAME} cmd/api/main.go

# Runtime stage
FROM alpine:3.22 AS runtime

# Re-declare args for this stage
ARG PROJECT_NAME
ARG APP_USER_ID=1001
ARG APP_GROUP_ID=1001
ARG TZ=America/Belem

ENV APP_NAME=${PROJECT_NAME}
ENV TZ=${TZ}
ENV APP_HOME=/${APP_NAME}
ENV LOG_DIR=/var/log/${APP_NAME}

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata curl wget bash

# Set timezone
RUN ln -snf /usr/share/zoneinfo/${TZ} /etc/localtime && \
    echo ${TZ} > /etc/timezone

# Create non-root user and group
RUN addgroup -g ${APP_GROUP_ID} ${APP_NAME} && \
    adduser -D -u ${APP_USER_ID} -G ${APP_NAME} -s /bin/sh ${APP_NAME} && \
    mkdir -p ${APP_HOME} ${LOG_DIR} && \
    chown -R ${APP_NAME}:${APP_NAME} ${APP_HOME} ${LOG_DIR}

# Copy binary from build stage
COPY --from=build --chown=${APP_NAME}:${APP_NAME} /build/${APP_NAME} /usr/local/bin/${APP_NAME}

# Ensure binary is executable
RUN chmod +x /usr/local/bin/${APP_NAME}

# Switch to non-root user
USER ${APP_NAME}

WORKDIR ${APP_HOME}

# Define volumes
VOLUME ["${APP_HOME}", "${LOG_DIR}"]

# Healthcheck configuration
# HEALTHCHECK --interval=60s --timeout=10s --start-period=5s --retries=3 \
#     CMD if [ -f /usr/local/bin/healthcheck.sh ]; then \
#             /usr/local/bin/healthcheck.sh; \
#         else \
#             /usr/local/bin/${APP_NAME} health || exit 1; \
#         fi

# Expose application port
# EXPOSE 3000

# Entrypoint
ENTRYPOINT ["/bin/sh", "-c", "/usr/local/bin/${APP_NAME} -config ${APP_HOME}/configs/config.yml"]
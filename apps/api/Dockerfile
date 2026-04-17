ARG GO_BUILD_BASE_IMAGE=1.25.7-alpine3.23
ARG GO_BUILD_FLAGS=""
ARG PROJECT_NAME="helix-api"
ARG TZ=America/Belem

FROM golang:${GO_BUILD_BASE_IMAGE} AS build

ARG PROJECT_NAME
ARG GO_BUILD_FLAGS
ARG TZ

ENV APP_NAME=${PROJECT_NAME}
ENV TZ=${TZ}

WORKDIR /build

RUN apk add --no-cache git gcc musl-dev tzdata

RUN ln -snf /usr/share/zoneinfo/${TZ} /etc/localtime && \
    echo ${TZ} > /etc/timezone

# Copy workspace and module files
COPY go.work go.work.sum ./
COPY apps/api/go.mod apps/api/go.sum ./apps/api/
COPY packages/config/go.mod packages/config/go.sum ./packages/config/
COPY packages/logger/go.mod packages/logger/go.sum ./packages/logger/
COPY packages/auth/go.mod packages/auth/go.sum ./packages/auth/
COPY packages/device/go.mod packages/device/go.sum ./packages/device/
COPY packages/datamodel/go.mod packages/datamodel/go.sum ./packages/datamodel/
COPY packages/task/go.mod packages/task/go.sum ./packages/task/

RUN go work sync

# Copy full source
COPY apps/api/ ./apps/api/
COPY packages/config/ ./packages/config/
COPY packages/logger/ ./packages/logger/
COPY packages/auth/ ./packages/auth/
COPY packages/device/ ./packages/device/
COPY packages/datamodel/ ./packages/datamodel/
COPY packages/task/ ./packages/task/

RUN CGO_ENABLED=1 go build ${GO_BUILD_FLAGS} -o ${APP_NAME} ./apps/api/cmd/...

FROM alpine:3.22 AS runtime

ARG PROJECT_NAME
ARG APP_USER_ID=1001
ARG APP_GROUP_ID=1001
ARG TZ=America/Belem

ENV APP_NAME=${PROJECT_NAME}
ENV TZ=${TZ}
ENV APP_HOME=/${APP_NAME}
ENV LOG_DIR=/var/log/${APP_NAME}

RUN apk add --no-cache ca-certificates tzdata curl wget bash

RUN ln -snf /usr/share/zoneinfo/${TZ} /etc/localtime && \
    echo ${TZ} > /etc/timezone

RUN addgroup -g ${APP_GROUP_ID} ${APP_NAME} && \
    adduser -D -u ${APP_USER_ID} -G ${APP_NAME} -s /bin/sh ${APP_NAME} && \
    mkdir -p ${APP_HOME} ${LOG_DIR} && \
    chown -R ${APP_NAME}:${APP_NAME} ${APP_HOME} ${LOG_DIR}

COPY --from=build --chown=${APP_NAME}:${APP_NAME} /build/${APP_NAME} /usr/local/bin/${APP_NAME}

RUN chmod +x /usr/local/bin/${APP_NAME}

USER ${APP_NAME}

WORKDIR ${APP_HOME}

VOLUME ["${APP_HOME}", "${LOG_DIR}"]

ENTRYPOINT ["/bin/sh", "-c", "/usr/local/bin/${APP_NAME} -config ${APP_HOME}/configs/config.yml"]

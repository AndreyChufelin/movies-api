FROM golang:1.23.2 AS base
FROM base AS build

ENV BIN_FILE=/opt/movies/api
ENV CODE_DIR=/go/src/

WORKDIR ${CODE_DIR}

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . ${CODE_DIR}

ARG LDFLAGS
RUN CGO_ENABLED=0 go build \
        -ldflags "$LDFLAGS" \
        -o ${BIN_FILE} cmd/api/*

FROM alpine:3.9 AS prod

ENV BIN_FILE="/opt/movies/api"
COPY --from=build ${BIN_FILE} ${BIN_FILE}

CMD ["sh", "-c", "${BIN_FILE}"]

FROM base AS dev

WORKDIR /app

RUN go install github.com/air-verse/air@latest

COPY go.mod go.sum ./
RUN go mod download

CMD ["air", "-c", ".air.toml"]

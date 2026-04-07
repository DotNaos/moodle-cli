FROM golang:1.24-bookworm AS builder

WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build -o /out/moodle ./cmd/moodle

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /app
ENV MOODLE_CLI_HOME=/data
VOLUME ["/data"]
COPY --from=builder /out/moodle /usr/local/bin/moodle
EXPOSE 8080
ENTRYPOINT ["moodle"]
CMD ["--help"]

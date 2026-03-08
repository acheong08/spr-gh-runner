FROM golang:1.25.6 AS builder
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY pkg ./pkg
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/test-generator ./cmd/test-generator

FROM ubuntu:24.04
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    docker.io \
    wget \
    tar \
    nodejs \
    npm \
    sudo \
    bash \
 && rm -rf /var/lib/apt/lists/*
WORKDIR /workspace
COPY --from=builder /out/test-generator /usr/local/bin/test-generator
COPY templates /opt/spr-gh-runner/templates
ENV TEMPLATES_DIR=/opt/spr-gh-runner/templates
ENTRYPOINT ["/bin/bash"]

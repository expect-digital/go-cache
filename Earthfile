VERSION 0.7

ARG --global go_version=1.18.10
ARG --global golangci_lint_version=1.54.2

go:
  FROM golang:$go_version-alpine
  ENV CGO_ENABLED=0
  WORKDIR /lru
  COPY --dir internal lru .
  COPY go.mod go.sum .
  RUN --mount=type=cache,target=/go/pkg/mod go mod download

  SAVE ARTIFACT /lru


lint:
  FROM golangci/golangci-lint:v$golangci_lint_version-alpine
  WORKDIR /lru
  COPY +go/lru .

  RUN --mount=type=cache,target=/root/.cache/golangci_lint golangci-lint run --timeout 3m

test:
  FROM +go
  RUN --mount=type=cache,target=/go/pkg/mod go test ./... -count 10


check:
  BUILD +lint
  BUILD +test

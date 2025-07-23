VERSION 0.8
ARG go_version=1.24.5
FROM golang:$go_version-alpine

src:
  ENV CGO_ENABLED=0
  WORKDIR /src
  COPY --dir internal lru .
  COPY go.mod go.sum .
  RUN \
    --mount=type=cache,id=go-mod,target=/go/pkg/mod \
    go mod download
  SAVE ARTIFACT /src

lint:
  ARG golangci_lint_version=2.3.0
  FROM golangci/golangci-lint:v$golangci_lint_version-alpine
  WORKDIR /src
  COPY .golangci.yml .
  COPY --dir +src/src /
  RUN \
    --mount=type=cache,id=go-mod,target=/go/pkg/mod \
    --mount=type=cache,id=go-build,target=/root/.cache/go-build \
    --mount type=cache,id=golangci,target=/root/.cache/golangci_lint \
    golangci-lint run --timeout 3m -v

test:
  FROM +src
  RUN \
    --mount=type=cache,id=go-mod,target=/go/pkg/mod \
    --mount=type=cache,id=go-build,target=/root/.cache/go-build \
    go test ./... -count 10

check:
  BUILD +lint
  BUILD +test

VERSION 0.8
# renovate: datasource=docker depName=golang
ARG go_version=1.26.6-alpine3.24@sha256:3889b425f035be855a72fb4755265311293b6d414521f0a519d819df32222d83
FROM golang:$go_version
WORKDIR /src

src:
  ENV CGO_ENABLED=0
  WORKDIR /src
  COPY --dir internal lru .
  COPY go.mod go.sum .
  RUN \
    --mount=type=cache,id=go-mod,target=/go/pkg/mod \
    go mod download
  SAVE ARTIFACT /src

# lint runs all linters for golang
lint:
  # renovate: datasource=docker depName=golangci/golangci-lint
  ARG golangci_lint_version=v2.13.1-alpine@sha256:f5e7bd15e2dce6f78f976acc07075f3208ce1a39b78f245f1ea984b2a39d105c
  FROM golangci/golangci-lint:$golangci_lint_version
  WORKDIR /src
  COPY .golangci.yml .
  COPY --dir +src/src /
  RUN \
    --mount=type=cache,id=go-mod,target=/go/pkg/mod \
    --mount=type=cache,id=go-build,target=/root/.cache/go-build \
    --mount type=cache,id=golangci,target=/root/.cache/golangci_lint \
    golangci-lint run --timeout 3m

# test runs unit tests
test:
  FROM +src
  RUN \
    --mount=type=cache,id=go-mod,target=/go/pkg/mod \
    --mount=type=cache,id=go-build,target=/root/.cache/go-build \
    go test ./... -count 10

# govulncheck checks golang vulnerabilities
govulncheck:
  RUN apk add git
  # renovate: datasource=go depName=golang.org/x/vuln/cmd/govulncheck
  ARG govulncheck_version=v1.1.4
  RUN go install golang.org/x/vuln/cmd/govulncheck@$govulncheck_version
  COPY --dir +src/src /
  RUN govulncheck ./...

# check verifies code quality by running linters and tests
check:
  BUILD +lint
  BUILD +test
  BUILD +govulncheck

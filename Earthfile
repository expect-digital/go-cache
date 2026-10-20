VERSION 0.8
# renovate: datasource=docker depName=golang
ARG go_version=1.27.3-alpine3.24@sha256:f2c872bcee949b7be9c490efd02b48a8a2fbf947cdf533f1c58525ec9d8170fd
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
  ARG golangci_lint_version=v2.15.0-alpine@sha256:2ba875b990f1d8501ad4ec19be1ea0701e316ec3a4b764f7b129f62f01318bbf
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
  ARG govulncheck_version=v1.7.0
  RUN go install golang.org/x/vuln/cmd/govulncheck@$govulncheck_version
  COPY --dir +src/src /
  RUN govulncheck ./...

# check verifies code quality by running linters and tests
check:
  BUILD +lint
  BUILD +test
  BUILD +govulncheck

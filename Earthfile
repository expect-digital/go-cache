VERSION 0.8
# renovate: datasource=docker depName=golang
ARG go_version=1.27.1-alpine3.24@sha256:cf6fca6641884b8433441b2b0652976f975e1d0fdd26d177eaaf8596087f3125
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
  ARG golangci_lint_version=v2.13.2-alpine@sha256:da1f79b184ff1d98781648407d05a99b9cba7b282aa3416b0a0eac102a3b0557
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

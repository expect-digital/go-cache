VERSION 0.8
# renovate: datasource=docker depName=golang
ARG go_version=1.27.4-alpine3.24@sha256:70bd7ddea8b6cbe5d6943cde279d6b67feb10cc924fb9100d1d8757c1575b280
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

# zizmor audits GitHub Actions workflows
zizmor:
  # renovate: datasource=docker depName=ghcr.io/zizmorcore/zizmor
  ARG zizmor_version=1.32.0@sha256:3014cc8f68f663935f69b3c3e0e41f5ce102b268d97db13db0f62208747f860d
  FROM ghcr.io/zizmorcore/zizmor:$zizmor_version
  WORKDIR /src
  COPY --dir .github .
  RUN zizmor --offline --strict-collection .github

# check verifies code quality by running linters and tests
check:
  BUILD +lint
  BUILD +test
  BUILD +govulncheck
  BUILD +zizmor

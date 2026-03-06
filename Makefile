SHELL := /bin/sh

GO ?= go
GOFMT ?= gofumpt
GOLANGCI_LINT ?= golangci-lint
BINARY_NAME ?= git-ho
BINARY_PATH ?= $(CURDIR)/dist/$(BINARY_NAME)
GOCACHE ?= $(CURDIR)/.cache/go-build
GOFILES = $$(find . -name '*.go' -type f -not -path './.cache/*' -not -path './dist/*' -not -path './.ho/*' | sort)
TEST_PKGS = $$( $(GO) list -f '{{if or .TestGoFiles .XTestGoFiles}}{{.ImportPath}}{{end}}' ./... | sed '/^$$/d' )

.PHONY: help build lint test fmt fmt-check tidy clean check ci prerelease_for_tagpr

help:
	@printf '%s\n' \
		'Available targets:' \
		'  make build  - build git-ho into ./dist' \
		'  make lint   - run golangci-lint for non-test packages' \
		'  make test   - run shuffled tests with coverage output' \
		'  make fmt    - run go fix ./... and gofumpt -w' \
		'  make fmt-check - fail if go fix or gofumpt would change files' \
		'  make check  - run lint, test, fmt-check' \
		'  make ci     - CI entrypoint (same as check)' \
		'  make prerelease_for_tagpr - prepare version/go files before tagpr release' \
		'  make tidy   - run go mod tidy with local GOCACHE' \
		'  make clean  - remove local build artifacts'

build:
	@mkdir -p "$(dir $(BINARY_PATH))" "$(GOCACHE)"
	GOCACHE="$(GOCACHE)" $(GO) build -o "$(BINARY_PATH)" .

lint:
	@mkdir -p "$(GOCACHE)"
	GOCACHE="$(GOCACHE)" "$(GOLANGCI_LINT)" run --tests=false ./...

test:
	$(GO) test $(TEST_PKGS) -shuffle=on -coverprofile=coverage.out -covermode=count -count=1

fmt:
	@mkdir -p "$(GOCACHE)"
	GOCACHE="$(GOCACHE)" $(GO) fix ./...
	"$(GOFMT)" -w $(GOFILES)

fmt-check:
	@mkdir -p "$(GOCACHE)"
	@fix_diff="$$(GOCACHE="$(GOCACHE)" $(GO) fix -diff ./...)"; \
	if [ -n "$$fix_diff" ]; then \
		printf '%s\n' "$$fix_diff"; \
		exit 1; \
	fi
	@unformatted="$$( "$(GOFMT)" -l $(GOFILES) )"; \
	if [ -n "$$unformatted" ]; then \
		printf '%s\n' "$$unformatted"; \
		exit 1; \
	fi

check: lint test fmt-check

ci: check

tidy:
	@mkdir -p "$(GOCACHE)"
	GOCACHE="$(GOCACHE)" $(GO) mod tidy

prerelease_for_tagpr: tidy
	git add go.mod go.sum version/version.go

clean:
	rm -rf "$(CURDIR)/dist" "$(CURDIR)/.cache"

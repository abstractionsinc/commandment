.DEFAULT_GOAL := help

GOLANGCI_LINT := go tool -modfile=tools.mod golangci-lint
LINT_REVIEW_LINTERS := cyclop,funlen,gocognit,maintidx
COMPLETION_TEST_TIMEOUT ?= 5m

.PHONY: help
help:
	@echo Commandment development targets
	@echo fmt - format Go source
	@echo lint - run the authoritative lint gate
	@echo lint-review - run non-gating review linters
	@echo test - run the complete ordinary test suite
	@echo test-conformance - run CLI conformance tests
	@echo test-coverage - report owner-scoped coverage
	@echo test-race - run the broad race campaign
	@echo bench - screen the permanent CLI benchmark suite
	@echo tidy - tidy the main and tooling modules
	@echo prepare-shell-runtimes - prepare repository-managed shell runtimes
	@echo test-shell-completion - run real-shell completion integration tests

.PHONY: fmt lint lint-review
fmt:
	$(GOLANGCI_LINT) fmt

lint:
	$(GOLANGCI_LINT) config verify
	$(GOLANGCI_LINT) fmt --diff
	$(GOLANGCI_LINT) run

lint-review:
	$(GOLANGCI_LINT) config verify
	$(GOLANGCI_LINT) run --enable-only=$(LINT_REVIEW_LINTERS) --issues-exit-code=0

.PHONY: test test-conformance test-coverage test-race
test:
	go test -count=1 -p=1 ./...

test-conformance:
	go test -count=1 -p=1 ./cli/test/conformance/...

test-coverage: SHELL := /bin/sh
test-coverage:
	@set -eu; coverage_dir="$$(mktemp -d "$${TMPDIR:-/tmp}/commandment-coverage.XXXXXX")"; \
	trap 'rm -rf -- "$${coverage_dir}"' 0; \
	go test -count=1 -p=1 -covermode=set -coverpkg=./cli -coverprofile="$${coverage_dir}/cli.out" ./cli ./cli/test/conformance/... ./cli/test/integration/completion; \
	go test -count=1 -covermode=set -coverprofile="$${coverage_dir}/clitest.out" ./clitest; \
	go test -count=1 -covermode=set -coverprofile="$${coverage_dir}/clibench.out" ./clibench; \
	go test -count=1 -covermode=set -coverprofile="$${coverage_dir}/shelltest.out" ./shelltest; \
	for package in cli clitest clibench shelltest; do \
		printf '\n%s\n' "$${package}"; \
		go tool cover -func="$${coverage_dir}/$${package}.out"; \
	done

test-race:
	go test -count=1 -p=1 -race ./...

.PHONY: bench
bench:
	go test -run="^$$" -bench=. -benchmem -benchtime=50ms -count=3 ./cli/test/benchmark

.PHONY: tidy
tidy:
	go mod tidy
	go mod tidy -modfile=tools.mod

.PHONY: prepare-shell-runtimes test-shell-completion
prepare-shell-runtimes: GOHOSTOS = $(shell go env GOHOSTOS)
prepare-shell-runtimes:
	@$(if $(filter windows,$(GOHOSTOS)),echo No repository-managed shell runtimes are required on Windows.,GOHOSTOS=$(GOHOSTOS) ./scripts/shells/prepare-runtimes.sh)

test-shell-completion:
	go test -count=1 -timeout=$(COMPLETION_TEST_TIMEOUT) ./cli/test/integration/completion -args -integration $(if $(CASE),-case=$(CASE))

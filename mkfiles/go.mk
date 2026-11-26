# Find repository-owned Go modules, excluding external checkouts.
GO_MODULES := $(shell find . \( -path './api' -o -path './services' \) -prune -o -name go.mod -exec dirname {} \; | sort)

.PHONY: go-build
go-build:
	@for mod in $(GO_MODULES); do \
       	(cd $$mod && go generate ./... && go mod tidy && go build ./...) || exit $$?; \
	done;

.PHONY: go-fmt
go-fmt:
	@for mod in $(GO_MODULES); do \
		(cd $$mod && golangci-lint fmt ./...) || exit $$?; \
	done;

.PHONY: go-lint
go-lint:
	@for mod in $(GO_MODULES); do \
		(cd $$mod && golangci-lint fmt ./... --diff && golangci-lint run ./...) || exit $$?; \
	done;

# Set to a seed printed by go test to reproduce the test order.
GO_TEST_SHUFFLE ?= on

.PHONY: go-test go-test-race
go-test:
	@for mod in $(GO_MODULES); do \
		(cd $$mod && go test -shuffle="$(GO_TEST_SHUFFLE)" -cover $(GO_TEST_FLAGS) ./...) || exit $$?; \
	done;

go-test-race: GO_TEST_FLAGS := -race
go-test-race: go-test

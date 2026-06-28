all: setup build

include ./mkfiles/buf.mk
include ./mkfiles/go.mk

.PHONY: setup
setup:
	@aqua i && mise i

.PHONY: generate
generate: buf-generate
	@sqlc generate

.PHONY: build
build: generate go-build

.PHONY: fmt
fmt: buf-fmt go-fmt

.PHONY: lint
lint: buf-lint go-lint

.PHONY: test
test: go-test migration-test

.PHONY: migration-test
migration-test:
	@./tests/migration-test.sh

.PHONY: run
run:
	@cd apphosts && aspire run

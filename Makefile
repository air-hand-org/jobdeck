all: setup build

include ./mkfiles/*.mk

.PHONY: setup
setup:
	@aqua i && mise i

.PHONY: generate
generate: buf-generate
	@sqlc generate

.PHONY: build
build: generate go-build cs-build

.PHONY: fmt
fmt: buf-fmt go-fmt cs-fmt

.PHONY: lint
lint: buf-lint go-lint cs-lint

.PHONY: check-clean
check-clean:
	@git status --short
	@test -z "$$(git status --porcelain)"

.PHONY: test
test: go-test migration-test cs-test

.PHONY: migration-test
migration-test:
	@./tests/migration-test.sh

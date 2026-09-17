all: setup build

include ./mkfiles/*.mk

.PHONY: setup
setup:
	@aqua i && mise i

.PHONY: build
build: go-build cs-build

.PHONY: fmt
fmt: go-fmt cs-fmt tofu-fmt

.PHONY: lint
lint: go-lint cs-lint

.PHONY: check-clean
check-clean:
	@git status --short
	@test -z "$$(git status --porcelain)"

.PHONY: test
test: go-test cs-test

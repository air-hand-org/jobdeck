.PHONY: buf-generate
buf-generate:
	@buf build \
	&& buf generate --clean

.PHONY: buf-lint
buf-lint:
	@buf format --diff --exit-code && buf lint

.PHONY: buf-fmt
buf-fmt:
	@buf format -w

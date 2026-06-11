.PHONY: buf-generate
buf-generate:
	@buf build \
	&& buf generate --clean

.PHONY: buf-lint
buf-lint:
	@buf lint

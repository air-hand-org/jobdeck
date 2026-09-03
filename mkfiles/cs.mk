CS_MK := $(lastword $(MAKEFILE_LIST))
CS_MK_DIR := $(dir $(abspath $(CS_MK)))
CS_SLN := Jobdeck.slnx

.PHONY: cs-restore
cs-restore:
	@cd $(CS_MK_DIR)../ && \
	    dotnet restore $(CS_SLN)

.PHONY: cs-build
cs-build: cs-restore
	@cd $(CS_MK_DIR)../ && \
	    dotnet build $(CS_SLN) --no-restore

.PHONY: cs-test-prepare
cs-test-prepare:
	@bash $(CS_MK_DIR)../tests/prepare-e2e.sh

.PHONY: cs-test
cs-test: cs-build cs-test-prepare
	@cd $(CS_MK_DIR)../ && \
	    dotnet test $(CS_SLN) --no-build --no-restore

.PHONY: cs-lint
cs-lint:
	@cd $(CS_MK_DIR)../ && \
	    dotnet format --no-restore --verify-no-changes

.PHONY: cs-fmt
cs-fmt:
	@cd $(CS_MK_DIR)../ && \
	    dotnet format --no-restore

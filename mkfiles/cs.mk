CS_MK := $(lastword $(MAKEFILE_LIST))
CS_MK_DIR := $(dir $(abspath $(CS_MK)))
CS_SLN := Jobdeck.slnx

.PHONY: cs-build
cs-build:
	@cd $(CS_MK_DIR)../ && \
	    dotnet build $(CS_SLN)

.PHONY: cs-test-prepare
cs-test-prepare:
	@pwsh $(CS_MK_DIR)../tests/e2e/bin/Debug/net10.0/playwright.ps1 install chromium

.PHONY: cs-test
cs-test: cs-build cs-test-prepare
	@cd $(CS_MK_DIR)../ && \
	    dotnet test $(CS_SLN) --no-build --no-restore

.PHONY: cs-lint
cs-lint:
	@cd $(CS_MK_DIR)../ && \
	    dotnet format --exclude ./gen/csharp/ --verify-no-changes

.PHONY: cs-fmt
cs-fmt:
	@cd $(CS_MK_DIR)../ && \
	    dotnet format --exclude ./gen/csharp/

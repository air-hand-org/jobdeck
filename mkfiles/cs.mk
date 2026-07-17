CS_MK := $(lastword $(MAKEFILE_LIST))
CS_MK_DIR := $(dir $(abspath $(CS_MK)))
CS_PRJS := $(shell find $(CS_MK_DIR)../. -name *.csproj -exec dirname {} \; | sort)

.PHONY: cs-build
cs-build:
	@for prj in $(CS_PRJS); do \
		(cd $$prj && dotnet build) || exit $$?; \
	done;

.PHONY: cs-test
cs-test:
	@for prj in $(CS_PRJS); do \
		(cd $$prj && dotnet test) || exit $$?; \
	done;

.PHONY: cs-lint
cs-lint:
	@cd $(CS_MK_DIR)../ && \
	    dotnet format --exclude ./gen/csharp/ --verify-no-changes

.PHONY: cs-fmt
cs-fmt:
	@cd $(CS_MK_DIR)../ && \
	    dotnet format --exclude ./gen/csharp/

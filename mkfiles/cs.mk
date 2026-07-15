CS_PRJS := $(shell find . -name *.csproj -exec dirname {} \; | sort)

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

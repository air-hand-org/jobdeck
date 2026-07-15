ASPIRE_MK := $(lastword $(MAKEFILE_LIST))
ASPIRE_MK_DIR := $(dir $(abspath $(ASPIRE_MK)))
APPHOSTS_CSPROJ := $(ASPIRE_MK_DIR)../apphosts/Apphosts.csproj

.PHONY: run
run: export DEBUG ?= false # (true/false)
run:
	@aspire run --apphost $(APPHOSTS_CSPROJ)

.PHONY: watch-ui
watch-ui:
	watchexec \
		--watch $(ASPIRE_MK_DIR)../services/ui \
		--exts cs,razor,csproj \
		--delay-run 3s \
		aspire resource ui restart --apphost $(APPHOSTS_CSPROJ)

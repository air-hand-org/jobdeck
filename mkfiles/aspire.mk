ASPIRE_MK := $(lastword $(MAKEFILE_LIST))
ASPIRE_MK_DIR := $(dir $(abspath $(ASPIRE_MK)))
APPHOSTS_CSPROJ := $(ASPIRE_MK_DIR)../apphosts/Apphosts.csproj

.PHONY: run
run: ghcr-login
run: export DEBUG ?= false # (true/false)
run:
	@aspire run --apphost $(APPHOSTS_CSPROJ)

.PHONY: ghcr-login
ghcr-login:
	@test -n "$${GITHUB_PACKAGES_TOKEN:-}" || { echo "GITHUB_PACKAGES_TOKEN must be set in .env.local to pull private GHCR images." >&2; exit 1; }
	@test -n "$${GITHUB_ACTOR:-}" || { echo "GITHUB_ACTOR must be set to log in to GHCR." >&2; exit 1; }
	@printf '%s' "$$GITHUB_PACKAGES_TOKEN" | docker login ghcr.io --username "$$GITHUB_ACTOR" --password-stdin

.PHONY: watch-ui
watch-ui:
	watchexec \
		--watch $(ASPIRE_MK_DIR)../services/ui \
		--exts cs,razor,csproj \
		--delay-run 3s \
		aspire resource ui restart --apphost $(APPHOSTS_CSPROJ)

SERVICES_MK := $(lastword $(MAKEFILE_LIST))
SERVICES_MK_DIR := $(dir $(abspath $(SERVICES_MK)))

.PHONY: services-build
services-build:
	$(MAKE) -C services/auth setup
	cd services/auth && mise exec -- $(MAKE) build
	$(MAKE) -C services/job-api setup
	cd services/job-api && mise exec -- $(MAKE) build
	$(MAKE) -C services/job-worker setup
	cd services/job-worker && mise exec -- $(MAKE) build
	$(MAKE) -C services/ui setup
	cd services/ui && mise exec -- $(MAKE) build
	$(MAKE) -C services/demo setup
	cd services/demo && mise exec -- $(MAKE) build

.PHONY: clone-service-repo
clone-service-repo:
	@bash "$(SERVICES_MK_DIR)../tests/clone-service-repo.sh"

.PHONY: test-clone-service-repo
test: test-clone-service-repo
test-clone-service-repo:
	@bash "$(SERVICES_MK_DIR)../tests/clone-service-repo-test.sh"

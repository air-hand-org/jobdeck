SERVICES_MK := $(lastword $(MAKEFILE_LIST))
SERVICES_MK_DIR := $(dir $(abspath $(SERVICES_MK)))

.PHONY: services-build
services-build:
	$(MAKE) -C services/auth build
	$(MAKE) -C services/job-api build
	$(MAKE) -C services/job-worker build
	$(MAKE) -C services/ui build
	$(MAKE) -C services/demo build

.PHONY: clone-service-repo
clone-service-repo:
	@bash "$(SERVICES_MK_DIR)../tests/clone-service-repo.sh"

.PHONY: test-clone-service-repo
test: test-clone-service-repo
test-clone-service-repo:
	@bash "$(SERVICES_MK_DIR)../tests/clone-service-repo-test.sh"

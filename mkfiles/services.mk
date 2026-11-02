SERVICES_MK := $(lastword $(MAKEFILE_LIST))
SERVICES_MK_DIR := $(dir $(abspath $(SERVICES_MK)))

.PHONY: services-build
services-build:
	# Keep setup before build even when the parent make runs in parallel.
	cd services/auth && mise exec -- $(MAKE) -j1 setup build
	cd services/job-api && mise exec -- $(MAKE) -j1 setup build
	cd services/job-worker && mise exec -- $(MAKE) -j1 setup build
	cd services/ui && mise exec -- $(MAKE) -j1 setup build
	cd services/demo && mise exec -- $(MAKE) -j1 setup build

.PHONY: clone-service-repo
clone-service-repo:
	@bash "$(SERVICES_MK_DIR)../tests/clone-service-repo.sh"

.PHONY: test-clone-service-repo
test: test-clone-service-repo
test-clone-service-repo:
	@bash "$(SERVICES_MK_DIR)../tests/clone-service-repo-test.sh"

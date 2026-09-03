CONTAINER_MK := $(lastword $(MAKEFILE_LIST))
CONTAINER_MK_DIR := $(dir $(abspath $(CONTAINER_MK)))

.PHONY: container-build
container-build: container-build-go container-build-cs container-build-other

.PHONY: container-build-go
container-build-go:
	@cd $(CONTAINER_MK_DIR)../ \
	&& GITHUB_CONTENTS_TOKEN="$$(gh auth token)" docker build \
		--secret id=github_contents_token,env=GITHUB_CONTENTS_TOKEN \
		-f containers/job-api/Dockerfile . \
	&& GITHUB_CONTENTS_TOKEN="$$(gh auth token)" docker build \
		--secret id=github_contents_token,env=GITHUB_CONTENTS_TOKEN \
		-f containers/auth/Dockerfile . \
	&& GITHUB_CONTENTS_TOKEN="$$(gh auth token)" docker build \
		--secret id=github_contents_token,env=GITHUB_CONTENTS_TOKEN \
		-f containers/job-worker/Dockerfile . \
	;

.PHONY: container-build-cs
container-build-cs:
	@cd $(CONTAINER_MK_DIR)../ \
	&& docker build \
		--build-arg GITHUB_ACTOR \
		--secret id=github_packages_token,env=GITHUB_PACKAGES_TOKEN \
		-f containers/ui/Dockerfile . \
	;

.PHONY: container-build-other
container-build-other:
	@cd $(CONTAINER_MK_DIR)../containers/forward-proxy \
	&& docker build . \
	&& cd $(CONTAINER_MK_DIR)../ \
	&& docker build \
		-f containers/demo/Dockerfile . \
	;

# Find repository-owned Go modules, excluding external checkouts.
GO_MODULES := $(shell find . \( -path './api' -o -path './services' \) -prune -o -name go.mod -exec dirname {} \; | sort)

.PHONY: go-work-use
go-work-use:
	@for mod in $(GO_MODULES); do \
		go work use $$mod || exit $$?; \
	done

.PHONY: go-build
go-build: go-work-use
	@for mod in $(GO_MODULES); do \
       	(cd $$mod && go generate ./... && go mod tidy && go build ./...) || exit $$?; \
	done;

.PHONY: go-fmt
go-fmt:
	@for mod in $(GO_MODULES); do \
		(cd $$mod && golangci-lint fmt ./...) || exit $$?; \
	done;

.PHONY: go-lint
go-lint:
	@for mod in $(GO_MODULES); do \
		(cd $$mod && golangci-lint fmt ./... --diff && golangci-lint run ./...) || exit $$?; \
	done;

.PHONY: go-test
go-test:
	@for mod in $(GO_MODULES); do \
       	(cd $$mod && go test -cover ./...) || exit $$?; \
	done;

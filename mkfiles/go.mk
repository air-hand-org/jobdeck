GO_MODULES := $(shell find . -name go.mod -exec dirname {} \; | sort)

.PHONY: go-work-use
go-work-use:
	@for mod in $(GO_MODULES); do \
    	(go work use $$mod); \
	done

.PHONY: go-build
go-build: go-work-use
	@for mod in $(GO_MODULES); do \
       	(cd $$mod && go generate ./... && go mod tidy && go build ./...) || exit $$?; \
	done;

.PHONY: go-fmt
go-fmt:
	@golangci-lint fmt ./...

.PHONY: go-lint
go-lint:
	@golangci-lint fmt ./... --diff \
	;
#	&& for mod in $(GO_MODULES); do \
#       	(cd $$mod && golangci-lint run ./...) || exit $$?; \
#	done;

.PHONY: go-test
go-test:
	@for mod in $(GO_MODULES); do \
       	(cd $$mod && go test -cover ./...) || exit $$?; \
	done;

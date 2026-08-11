TOFU_ROOTS := $(sort $(dir $(wildcard iac/*/versions.tf)))

.PHONY: tofu-fmt
tofu-fmt:
	@tofu fmt -recursive iac

.PHONY: tofu-lint
tofu-lint:
	@tofu fmt -check -recursive iac
	@for root in $(TOFU_ROOTS); do \
		(cd $$root && tofu init -backend=false -input=false && tofu validate -no-color) || exit $$?; \
	done

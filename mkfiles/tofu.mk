TOFU_ROOTS := $(sort $(dir $(wildcard iac/*/versions.tf)))

.PHONY: tofu-fmt
tofu-fmt:
	@tofu fmt -recursive iac

.PHONY: tofu-lint
tofu-lint:
	@tofu fmt -check -recursive iac

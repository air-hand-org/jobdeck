TOFU_ROOTS := $(sort $(dir $(wildcard iac/*/versions.tf)))

# Pre-generated inputs only: never let TerraVision initialize or plan a root.
# See iac/README.md for prerequisites and handling of private local outputs.
TOFU_DIAGRAM_SOURCE ?=
TOFU_DIAGRAM_PLAN ?=
TOFU_DIAGRAM_GRAPH ?=
TOFU_DIAGRAM_OUT ?=

.PHONY: tofu-diagram
tofu-diagram:
	@test -d "$(TOFU_DIAGRAM_SOURCE)" || { echo 'Set TOFU_DIAGRAM_SOURCE to an existing local source directory' >&2; exit 1; }
	@test -f "$(TOFU_DIAGRAM_PLAN)" || { echo 'Set TOFU_DIAGRAM_PLAN to saved plan JSON (not a binary plan)' >&2; exit 1; }
	@test -f "$(TOFU_DIAGRAM_GRAPH)" || { echo 'Set TOFU_DIAGRAM_GRAPH to saved graph DOT' >&2; exit 1; }
	@test -n "$(TOFU_DIAGRAM_OUT)" || { echo 'Set TOFU_DIAGRAM_OUT to a private absolute output prefix' >&2; exit 1; }
	@terravision draw --engine tofu --source "$(abspath $(TOFU_DIAGRAM_SOURCE))" \
		--planfile "$(abspath $(TOFU_DIAGRAM_PLAN))" --graphfile "$(abspath $(TOFU_DIAGRAM_GRAPH))" \
		--outfile "$(abspath $(TOFU_DIAGRAM_OUT))" --format png --use-tf-names

.PHONY: tofu-fmt
tofu-fmt:
	@tofu fmt -recursive iac

.PHONY: tofu-lint
tofu-lint:
	@tofu fmt -check -recursive iac

#!/usr/bin/env bash
set -euo pipefail

# Optional release tag and service names identify images built by this release.
# Independently released services can retain a different version in the manifest.
release_tag="${1:-}"
if (( $# > 0 )); then
  shift
fi

yq -o=json '.' "${PRODUCTION_MANIFEST:-environments/production.yaml}" |
  jq -ce --arg release_tag "$release_tag" --args '
    .images
    | if type != "object" then error("images must be an object") else . end
    | if (["auth", "ui", "demo", "job-api", "job-worker", "forward-proxy"] - keys | length) != 0
      then error("missing required service image") else . end
    | if (keys | map(gsub("-"; "_")) | unique | length) != length
      then error("service names must remain unique after normalization") else . end
    | if all(.[];
        (keys == ["version"]) and
        (.version | type == "string") and
        (.version | test("^[0-9]+\\.[0-9]+\\.[0-9]+$")))
      then . else error("each image must contain only an X.Y.Z version string") end
    | . as $images
    | if all($ARGS.positional[]; . as $service |
        $images[$service] != null and ("v" + $images[$service].version) == $release_tag)
      then . else error("published release tag must match each locally built image version") end
    | with_entries(.key |= gsub("-"; "_") | .value |= .version)
  ' "$@"

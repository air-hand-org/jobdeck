# Match iac/app/locals.tf: bootstrap owns registry host/repositories, while the
# manifest owns versions. Underscore service keys map back to hyphen image names.
if ($registry | test("^[a-z0-9][a-z0-9.-]*-docker\\.pkg\\.dev$")) and
   ($project | test("^[a-z][a-z0-9-]+[a-z0-9]$"))
then $versions else error("invalid Artifact Registry host or project ID") end
| with_entries(
    .key as $service
    | $repositories[$service] as $repository
    | if ($repository | type == "string") and
         ($repository | test("^[a-z][a-z0-9-]*$"))
      then . else error("missing or invalid bootstrap repository for " + $service) end
    | .value = ($registry + "/" + $project + "/" + $repository + "/" +
        ($service | gsub("_"; "-")) + ":v" + .value)
  )

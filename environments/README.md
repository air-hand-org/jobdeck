# Deployment images

`production.yaml` is the source of application image versions for image
publication and OpenTofu deployment. Each service declares only a string
`version` in `X.Y.Z` form. Registry locations are resolved by each consumer;
they are not deployment-version data. Artifact Registry enforces immutable tags.

Bootstrap owns registry creation, IAM, and registry settings. App IaC composes
`<host>/<project>/<registry>/<service>:v<version>` from bootstrap's
`artifact_registry_host` and `artifact_registry_repositories` remote state
outputs, the project ID, and this YAML. The publishing workflow obtains the same
outputs after GCP authentication. Non-publishing build CI validates the versions
without accessing GCP or bootstrap state.

This format is also intended for the future container-based Aspire composition;
the current AppHost still starts local service sources. Aspire will resolve its
repositories separately, such as through a local GHCR repository mapping, while
sharing versions from `production.yaml`. It will not need bootstrap state access
or a separately maintained version list. Publishing the same built image to both
registries and adding that mapping are deferred; see the
[local composition design](../docs/cloud-architecture.md#local-composition).

## Release ownership during extraction

All six images are still published by jobdeck in this change. Its release-please
configuration uses an explicit YAML JSONPath selector to update their versions
in the release PR. The YAML updater may reformat the file and remove comments;
keep explanatory text in this README.

On release publication, the build workflow reads this YAML from the release
checkout and requires each locally built image version to match the release
tag before publishing anything. OpenTofu reads the same YAML after image push
succeeds. PR plans also read the proposed versions; no placeholder tag is used.
A successful plan does not prove that the images exist yet: release PRs describe
images that will be built after merging.

For each service moved to independent publication:

1. Publish its image successfully from the new repository.
2. Remove it from jobdeck's release-please JSONPath selector, publishing steps,
   and the locally built service arguments in the production image reader step.
3. Update only its version through an image-update PR, retaining the other
   service versions. Accept notifications only after image push completes.

The `repository_dispatch` receiver and image-update PR deployment trigger will
be implemented in a separate PR. This change retains the existing release-event
deployment trigger and does not deploy every change to main.

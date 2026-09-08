# Deployment images

`production.yaml` is the source of application image references for image
publication and OpenTofu deployment. Each service declares a full `repository`
URI without a tag and a string `version` in `X.Y.Z` form. Consumers construct
`repository:vversion`. Artifact Registry enforces immutable tags.

This format is also intended for the future container-based Aspire composition;
the current AppHost still starts local service sources.

Bootstrap owns registry creation, IAM, and registry settings. Image repository
URIs are explicit here so consumers do not need access to bootstrap state.
OpenTofu still reads runtime identities and secret resource names from bootstrap
outputs. When changing a registry location or name, update its URI here too.

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

# Release-tooling references

The workflows and release design were checked against these official sources when this repository was assembled:

- Go release history: https://go.dev/doc/devel/release
- `actions/checkout`: https://github.com/actions/checkout
- `actions/setup-go`: https://github.com/actions/setup-go
- `actions/upload-artifact`: https://github.com/actions/upload-artifact
- `actions/download-artifact`: https://github.com/actions/download-artifact
- GitHub artifact attestations: https://github.com/actions/attest
- GitHub release management: https://docs.github.com/en/repositories/releasing-projects-on-github/managing-releases-in-a-repository
- `GITHUB_TOKEN`: https://docs.github.com/en/actions/concepts/security/github_token

Major action tags are used so Dependabot can propose compatible security and runtime updates. Review such updates before merging because GitHub-hosted and self-hosted runner requirements can change.

## Design influences

- Context compression (`acp compress`, `acp run`, `acp pack`, `acp expand`) follows the concept of [Headroom](https://github.com/headroomlabs-ai/headroom): compress tool output, logs, JSON and code before it reaches a model, keep errors and outliers, and keep originals locally so any elision can be retrieved. ACP's implementation is independent, deterministic, standard-library Go and uses no ML models or network proxy.

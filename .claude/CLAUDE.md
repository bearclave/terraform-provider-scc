# terraform-provider-scc

Documentation is in `README.md`. The organisation standards are in
`.claude/rules/` (managed copies from `bearclave/governance`). This file holds
only what is specific to this repository.

## This repository

- **Public** fork of terraprovider/terraform-provider-scc, published to the
  Terraform Registry as `bearclave/scc`. Nothing about a tenant goes here.
- **Generated code:** `internal/provider/*_resource.go` come from
  `go run ./cmd/gen-tf`; change the generator or the `*_override*.go` files.
- **Local checks:** `go vet ./...`, `go test ./...`, `go run ./cmd/gen-tf`
  leaves no diff, `cd tools && go generate ./...` for docs.
- **Releases:** the owner, `tools/release.sh vX.Y.Z` (signed tag, goreleaser,
  key from Key Vault). Never from a workflow.

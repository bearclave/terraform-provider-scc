# terraform-provider-scc

Terraform / OpenTofu provider for **Microsoft Purview / Security & Compliance**,
published as `bearclave/scc` on the Terraform Registry for
[`bearclave/iac-purview`](https://github.com/bearclave/iac-purview).

A permanent fork of [terraprovider/terraform-provider-scc](https://github.com/terraprovider/terraform-provider-scc),
maintained by Bearclave and not contributed upstream. It is regenerated on
tf-msadmin v0.9.5 and fixed so that sensitivity labels, label policies, DLP and
retention are managed in place
([upstream #11](https://github.com/terraprovider/terraform-provider-scc/issues/11)).
Licensed under the MIT licence of the upstream, see [LICENSE](LICENSE). Not
affiliated with or endorsed by Microsoft.

## How it fits

The provider drives the same Admin API as the `ExchangeOnlineManagement`
module's `Connect-IPPSSession` (Security & Compliance PowerShell). Resources
are **generated from the module's own cmdlet catalog** (via
[`tf-msadmin/genframework`](https://github.com/terraprovider/tf-msadmin) and the
[`go-exoscc`](https://github.com/terraprovider/go-exoscc) bindings), so the
surface stays in lock-step with the service.

```
terraform-provider-scc ──> bearclave/iac-purview   (Terraform Registry, bearclave/scc)
```

### Authentication

The configuration is aligned field-for-field with the `hashicorp/azuread` and
`azurerm` providers: the same attribute names, the same `ARM_*` / `AZURE_*`
environment variables, and every OIDC / workload-identity flavour. Supported
methods: client secret, client certificate (PEM or PKCS#12), OIDC federation,
Azure CLI, and managed identity.

```hcl
provider "scc" {
  tenant_id     = "00000000-0000-0000-0000-000000000000"
  client_id     = "11111111-1111-1111-1111-111111111111"
  client_secret = var.client_secret
  organization  = "contoso.onmicrosoft.com"
}
```

App-only auth requires the app to be granted `Exchange.ManageAsApp` and added to a
Security & Compliance role group; compliance routing additionally needs
`organization` (the tenant's routing domain), which is effectively required under
app-only.

## Layout

```
cmd/gen-tf/          the generator
internal/provider/   the provider; *_resource.go and *_data_source.go are generated
examples/            Terraform examples rendered into docs/
docs/                generated Registry documentation
tools/               release.sh and the tfplugindocs module
```

`internal/provider/*_resource.go` are generated (`DO NOT EDIT`); change the
generator or the `overrides*.go` files, not the output.

## How to make a change

```bash
git novy <type>/<slug>
go build ./...                           # build
go vet ./... && go test ./...            # checks
go run ./cmd/gen-tf                      # regenerate all resources from the catalog
go run ./cmd/gen-tf -noun ComplianceCase # regenerate one
cd tools && go generate ./...            # regenerate docs (tfplugindocs)
```

A pull request runs `security / scan`, `pr / check` and `pipeline / gate`: `go vet`,
`go test`, golangci-lint, and a check that generation and docs leave no diff.
Releases are cut by the owner with `tools/release.sh vX.Y.Z`.

## Pipeline

Calls `bearclave/ci` `security.yml` and `checks.yml`.

| Trigger | What runs | Identity |
|---|---|---|
| Pull request into `main` | `ci.yml`: `security.yml` and `checks.yml` (`go vet`, `go test`, generation and docs without diff, golangci-lint) | none |
| Pull request | `pr.yml`: the `pr` check of `bearclave/ci` | none |
| Weekly | `ci.yml`: the same, for new gitleaks rules over the old history | none |
| Release | `tools/release.sh` on the owner's Mac: tag, goreleaser, signed with the key from `kv-tech-sec-weu-001` | the owner |

## Conventions

Where the cmdlets do not fit the generic New/Get/Set/Remove mapping, the
generated resource is wrapped instead of edited:

- `cmd/gen-tf/overrides.go` lists the overridden nouns (labels, label
  policies, DLP and retention policies and rules). The generator detects lists
  that only `New-*` takes and `Set-*` changes through `-Add<X>`/`-Remove<X>`
  (labels, every `*_location`), makes them updatable in place, drops the
  `add_*`/`remove_*` companions and emits `zz_generated_overrides.go`.
- `internal/provider/overrides*.go` applies those list deltas on update,
  applies `Set-*`-only parameters (`Set-Label -Priority`) right after create,
  and on refresh keeps the configured value when the read-back is equivalent
  (`semantic.go`: JSON by content, case-insensitive enums and lists, locations
  by any identity, `LocaleSettings` by `localeKey` and locale).
- `internal/provider/readback.go` rewrites `Get-Label` / `Get-LabelPolicy`
  responses: `LabelActions` JSON and `Settings` "[key, value]" strings become
  the `ApplyContentMarking*`, `ApplyWaterMarking*`, `Encryption*` and
  `AdvancedSettings` properties the write parameters use.
- Every resource has schema version 1; state from 0.2.x (version 0) is
  converted by shape (`versioned.go`). Local development against
  sibling module checkouts uses the git-ignored `go.work`.

## Repository settings

Branch protection, required checks, merge settings and labels are
defined in [`bearclave/iac-github`](https://github.com/bearclave/iac-github)
(`infrastructure/repositories/terraform-provider-scc`). Engineering standards:
[`bearclave/governance`](https://github.com/bearclave/governance/tree/main/standards).

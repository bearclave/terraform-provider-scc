# terraform-provider-scc

[![Test](https://github.com/terraprovider/terraform-provider-scc/actions/workflows/test.yml/badge.svg)](https://github.com/terraprovider/terraform-provider-scc/actions/workflows/test.yml)
[![Build](https://github.com/terraprovider/terraform-provider-scc/actions/workflows/build.yml/badge.svg)](https://github.com/terraprovider/terraform-provider-scc/actions/workflows/build.yml)

A Terraform / OpenTofu provider for **Microsoft Purview / Security & Compliance**,
driven by the same Admin API the `ExchangeOnlineManagement` module's
`Connect-IPPSSession` (Security & Compliance PowerShell) uses. Resources are
**generated from the module's own cmdlet catalog** (via
[`tf-msadmin/genframework`](https://github.com/terraprovider/tf-msadmin) and the
[`go-exoscc`](https://github.com/terraprovider/go-exoscc) bindings), so the surface
stays in lock-step with the service.

> Not affiliated with or endorsed by Microsoft.

## Authentication

The provider's configuration is aligned field-for-field with the
`hashicorp/azuread` and `azurerm` providers — the same attribute names, the same
`ARM_*` / `AZURE_*` environment variables, and every OIDC / workload-identity
flavour (GitHub Actions, Azure DevOps, generic). Supported methods: client
secret, client certificate (PEM or PKCS#12), OIDC federation, Azure CLI, and
managed identity.

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

## Development

```bash
go build ./...                       # build
go test ./...                        # test
go run ./cmd/gen-tf                      # regenerate all resources from the catalog
go run ./cmd/gen-tf -noun ComplianceCase # regenerate one
cd tools && go generate ./...        # regenerate docs (tfplugindocs)
```

Resource files under `internal/provider/*_resource.go` are generated
(`DO NOT EDIT`); change the generator, not the output.

### Overrides

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
  by any identity).
- `internal/provider/readback.go` rewrites `Get-Label` / `Get-LabelPolicy`
  responses: `LabelActions` JSON and `Settings` "[key, value]" strings become
  the `ApplyContentMarking*`, `ApplyWaterMarking*`, `Encryption*` and
  `AdvancedSettings` properties the write parameters use.
- Every resource has schema version 1; state from 0.2.x (version 0) is
  converted by shape (`versioned.go`). Local development against
sibling module checkouts uses the git-ignored `go.work`.

## License

MIT — see [LICENSE](LICENSE).

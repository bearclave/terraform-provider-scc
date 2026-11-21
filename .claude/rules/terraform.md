---
paths:
  - "**/*.tf"
  - "**/.tflint.hcl"
---

# Terraform modules

<!-- Standard of bearclave/governance (standards/code/terraform.md). In other repositories this is a managed copy - change it in governance. -->

Modules live in `bearclave/iac-modules`; the live repositories call them
([terragrunt.md](https://github.com/bearclave/governance/blob/main/standards/code/terragrunt.md)). Skeleton:
[../templates/terraform-module/](https://github.com/bearclave/governance/tree/main/standards/templates/terraform-module/). Azure
rules (naming, tags, RBAC): [azure.md](https://github.com/bearclave/governance/blob/main/standards/code/azure.md).

## Layout of iac-modules

```text
iac-modules/
  docs/catalog.md      generated list of the modules
  azure/               azurerm, azapi
    _common/           internal building blocks: naming, principals, role-assignments, diagnostics, lock
    key-vault/  log-analytics-workspace/  resource-group/  storage-account/  ...
  entra/               azuread, msgraph
  intune/              terraprovider/microsoft365wp (Intune, Defender for Endpoint)
  exchange/            terraprovider/exo (Exchange Online, Defender for Office 365)
  purview/             terraprovider/scc
  github/              integrations/github
  routeros/            terraform-routeros/routeros
```

| Level | Rule |
|---|---|
| 1 | the platform the module manages (its provider), never a domain (`security/`, `monitoring/`) |
| 2 | one folder per module, flat |
| Name | the provider's resource type in kebab-case without the provider prefix (`azurerm_key_vault` -> `key-vault`); singular for one instance, plural when the module takes a map (`policy-assignments`, `teams`) |
| `_common/` | under its platform (`azure/_common/`); only code that two or more public modules call, never a unit |

The folder name of a module is the folder name of its units in the live
repositories ([terragrunt.md](https://github.com/bearclave/governance/blob/main/standards/code/terragrunt.md#folder-dictionary)).

## The rule without exception

**No variable has a `default`**, and no `optional(type, default)`. A value
every unit sets the same way is written once in the live repository
(`_common/units/<module>.hcl`), never hidden in the module. Opting out is an
explicit value (`{}`, `[]`, `"none"`, `null` where the variable is
`nullable`).

`optional(type)` without a default is allowed only for the fields of an
object that maps a list of provider items whose fields are optional by
nature (a firewall rule: `protocol`, `src_address`, ...); the module passes
`null` on and the provider leaves the field out.

## Files of a module

| File | Required | Content |
|---|---|---|
| `versions.tf` | yes | `required_version` (one value for the repository) and providers with a **lower bound** (`>= 4.0`); the live repository pins the exact version |
| `variables.tf` | yes | context first (`naming`, `name_override`, `resource_group_name`, `location`, `tags`), then the settings alphabetically, then the interfaces (`diagnostics`, `alerts`, `lock`, `role_assignments`) |
| `main.tf` | yes | the resources; past ~300 lines split into `<concern>.tf` (`rules.tf`) |
| `outputs.tf` | yes | every output with `description`; IDs and names; a secret only with `sensitive = true` |
| `locals.tf` | only when several files read the locals | otherwise `locals` at the top of the file that uses them |
| `README.md` | public modules | `## What it manages` (1-2 sentences, read by the catalog), `## Why this way` (provider gaps with the version seen, what is left out on purpose), `## Example` (the `inputs` of a unit); below them the block `<!-- BEGIN_TF_DOCS -->` ... `<!-- END_TF_DOCS -->` that terraform-docs writes from `.terraform-docs.yml` (requirements, inputs, outputs) |
| `tests/*.tftest.hcl` | in `_common/`; recommended where a module has logic; required for a collection module | `command = plan` with `mock_provider`, offline |
| `schema.json` | a module that takes a collection | the JSON Schema of one object of the collection (one `<key>.yaml`), 2020-12, closed (`additionalProperties: false`), with a `description` per key; the live repositories copy it to `.github/schemas/<module>.schema.json` ([terragrunt.md](https://github.com/bearclave/governance/blob/main/standards/code/terragrunt.md#schemas)) |
| `scripts/` | only for a provider gap | a script `local-exec` calls, with the issue it works around; never a script next to the `.tf` files |
| `.terraform.lock.hcl` | never | the lock belongs to the unit |
| `examples/` | no | the example is in the README; the units are the real ones |

`_common/` modules have no README and are never called from a unit; a unit
always calls a public module.

## Variables and outputs

- Every variable has `type` and `description` (English, at most three
  lines; the reasoning goes into the README). `nullable = false` where `null`
  makes no sense.
- **`type = any` is not allowed.** `diagnostics` and `alerts` are typed
  objects; `null` switches them off.
- `validation` for values the API would reject late (enums, formats,
  `lock = CanNotDelete | ReadOnly | none`).
- A module takes only the interfaces it implements. The names stay ours:
  `diagnostics`, `alerts`, `lock`, `role_assignments` (principals as
  `group:`, `user:`, `sp:`, `id:`).
- Every output has a `description`.

## Style

- `terraform fmt`; names `snake_case`; `this` for the single main resource,
  satellites by role (`lock`, `rbac`).
- Collections as `map(object)` keyed by a stable name, iterated with
  `for_each`; `count` only for a condition.
- Principals by display name (`group:`, `user:`, `sp:`), resolved by
  `_common/principals`; `id:` only for one the same change creates.
- `lifecycle.ignore_changes` only with a comment naming the process that owns
  the attribute.
- A provider gap is written in the README with the provider version it was
  seen in.
- No provider block and no backend in a module.
- Comments: [documentation.md](https://github.com/bearclave/governance/blob/main/standards/documentation.md#comments-in-code).

## Check blocks

A module may carry `check {}` blocks for assertions only the deployed
resource can answer: the site answers, a key exists, a setting the API
accepts but ignores took effect. Terraform evaluates them on every plan and
apply and only warns; the pipeline turns a failed one into a failed
verification after the apply (and a red drift run). Not for what a
`validation` or a `precondition` catches before the apply.

```hcl
check "site_answers" {
  data "http" "site" {
    url = "https://${azurerm_static_web_app.this.default_host_name}"
  }

  assert {
    condition     = data.http.site.status_code == 200
    error_message = "The site does not answer with 200."
  }
}
```

## Versions

- **One SemVer tag for the repository** (`vX.Y.Z`); no tag per module. A
  live repository pins it once (`modules_ref` in `account.hcl`).
- A unit may override the ref in its own `terraform { source }` only as a
  temporary canary, with an issue.
- A moved module path, a removed or renamed input, a changed type is a major
  release (`feat(<module>)!:`). A changed resource address inside a module
  gets a `moved` block in the module.

## Checks

| Check | Holds |
|---|---|
| `terraform fmt -check`, `terraform validate` | per module |
| `tflint` (managed `.tflint.hcl`) | recommended preset, azurerm ruleset, `terraform_documented_variables`, `terraform_documented_outputs`, `terraform_typed_variables`, `terraform_standard_module_structure`, `terraform_naming_convention`, `terraform_unused_declarations` |
| `trivy config` | HIGH and CRITICAL fail; an accepted finding in `.github/trivyignore.yaml` with the reason and a review date |
| `terraform test` | every module with `tests/` |
| terraform-docs | the README block of every module is current (`--output-check`, version in `.github/tool-versions.env`) |
| catalog | `docs/catalog.md` is what `bearclave/ci` `src/terraform/catalog.py` writes |
| conventions | `variable-no-default`, `variable-documented` |

Exceptions and the table of every check:
[pipelines.md](https://github.com/bearclave/governance/blob/main/standards/pipelines.md#checks-and-their-exceptions).

## Changing a module

A module change is its own pull request and release, merged before the
units that use it. The Impact section names the affected live repositories.

## Collection modules

A module that takes a collection of objects
([terragrunt.md](https://github.com/bearclave/governance/blob/main/standards/code/terragrunt.md#collections)):

- One variable, the collection: `map(object({...}))` keyed by the file's
  key, every field typed, `optional(type)` without a default for what an
  object may leave out. The module reads nothing from disk.
- `schema.json` describes the same object, key for key; a test checks that
  an example object passes through the variable, so that the two do not
  drift apart.
- Over a generic provider (`msgraph_resource`, `azapi_resource`) the module
  is named after the API type (`conditional-access-policies`), never after
  the resource; it validates what the API would refuse late (`validation`).
- References to other objects come in by name or by key (`group:<name>`,
  the key of a named location) and are resolved in the module, by a data
  source or a dependency's output.

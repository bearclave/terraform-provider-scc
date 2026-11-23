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
  modules/
    _empty/            the empty module a handed-over unit runs (removed blocks)
    azure/             azurerm, azapi
      _common/         internal building blocks: naming, principals, role-assignments, diagnostics, lock
      key-vault/  log-analytics-workspace/  resource-group/  storage-account/  ...
    entra/             azuread, msgraph
      _common/         principals, api-permissions
    intune/            terraprovider/microsoft365wp (Intune, Defender for Endpoint)
      _common/         assignments
    exchange/          terraprovider/exo (Exchange Online, Defender for Office 365)
    purview/           bearclave/scc (the organisation's fork of terraprovider/scc)
    github/            integrations/github
    routeros/          terraform-routeros/routeros
```

The top folder is `modules/` and nothing else but `docs/` and the root files
sits beside it (`layout.yaml`). The source of a module is
`<repository>//modules/<platform>/<module>`.

| Level | Rule |
|---|---|
| 1 | the platform the module manages (its provider), never a domain (`security/`, `monitoring/`) |
| 2 | one folder per module, flat |
| Name | the provider's resource type in kebab-case without the provider prefix (`azurerm_key_vault` -> `key-vault`); singular for one instance, plural when the module takes a map (`policy-assignments`, `teams`) |
| `_common/` | under the platform of the provider it uses (`entra/_common/principals`); only code that two or more modules call, by relative path from any platform (a platform's `_common/` may call another's); no README per block, always `tests/` |

- A module without a consumer is deleted in the next major release; a module
  for what [m365.md](https://github.com/bearclave/governance/blob/main/standards/code/m365.md#in-code-and-out-of-code) keeps out of code is not
  kept. A resource type is created by one module (the Logic App workflow is
  not also a Consumption module).
- The folder name of a module is the folder name of its units in the live
  repositories ([terragrunt.md](https://github.com/bearclave/governance/blob/main/standards/code/terragrunt.md#folder-dictionary)).

## The rule without exception

**No variable has a `default`**, and no `optional(type, default)`. A value
is never hidden in the module as a default: the unit declares it
([terragrunt.md](https://github.com/bearclave/governance/blob/main/standards/code/terragrunt.md#what-is-declared-where)). What a unit cannot be
allowed to switch off (purge protection, `shared_access_key_enabled = false`)
is not a variable at all but the module's baseline. Opting out is an
explicit value (`{}`, `[]`, `"none"`, `null` where the variable is
`nullable`).

`optional(type)` without a default is allowed only for the fields of an
object that maps a list of provider items whose fields are optional by
nature (a firewall rule: `protocol`, `src_address`, ...), and for the
optional arguments of an item the module forwards (`condition`,
`condition_version` of a role assignment; a rule of a GitHub ruleset that is
off when absent); the module passes `null` on and the provider leaves the
field out.

## Files of a module

| File | Required | Content |
|---|---|---|
| `versions.tf` | yes | `required_version` (one value for the repository) and providers with a **lower bound** (`>= 4.0`); the live repository pins the exact version |
| `variables.tf` | yes | context first: `naming`, `name_override` (or `name` where the unit folder is the name; outside Azure the object's own name: `name`, `login`), the scope (`management_group_id`, `subscription_id`, `resource_group_name`, `resource_group_id`, `workspace_id`), `location`, `tenant_id`, `tags`; then a collection variable; then the settings alphabetically; then the interfaces `diagnostics`, `alerts`, `lock`, `private_endpoints`, `role_assignments`. A unit's `inputs` follow this order ([azure.md](https://github.com/bearclave/governance/blob/main/standards/code/azure.md#interfaces-of-a-resource)) |
| `main.tf` | yes | the resources; past ~300 lines split into `<concern>.tf` (`rules.tf`) |
| `outputs.tf` | yes | every output with `description`; IDs and names; a secret only with `sensitive = true` |
| `locals.tf` | only when several files read the locals | otherwise `locals` at the top of the file that uses them |
| `README.md` | public modules | `## What it manages` (1-2 sentences, read by the catalog), `## Why this way` (only what is particular to the module, as bullets `Fixed:` (values set on purpose), `Left out:`, `Gap:` (provider, exact version, issue or `handbook/gotchas/` link); no history, no deployment status, no live object names, no rules of a live repository, no licence terms of a provider), `## Example` (the `inputs` of a unit with every input, the context as `include.root.locals.ctx.<name>`; a collection module shows one `<key>.yaml`), `## Reference` (the block `<!-- BEGIN_TF_DOCS -->` ... `<!-- END_TF_DOCS -->` that terraform-docs writes from `.terraform-docs.yml`) |
| `tests/*.tftest.hcl` | in `_common/`; recommended where a module has logic; required for a collection module | `command = plan` with `mock_provider`, offline; a test for every `validation` with `expect_failures`; a collection module passes its README example object through the variable, and that object is a fixture `collection-schema` validates, so that variable and schema cannot drift apart |
| `schema.json` | a module whose unit reads `definitions/<key>.yaml` files (a map written in HCL in the unit needs none) | the JSON Schema of one object (one `<key>.yaml`), 2020-12, closed (`additionalProperties: false`), no `$id`, a one-line `description` per key, every `validation` of the variable that JSON Schema can express (`enum`, `pattern`, `minimum`/`maximum`, `oneOf`, `uniqueItems`); the live repositories copy it to `.github/schemas/<module>.schema.json` ([terragrunt.md](https://github.com/bearclave/governance/blob/main/standards/code/terragrunt.md#schemas)) |
| `scripts/` | only for a provider gap | a script `local-exec` calls; its first line names the provider version and links the issue it works around; never a script next to the `.tf` files |
| `.terraform.lock.hcl` | never | the lock belongs to the unit |
| `examples/` | no | the example is in the README; the units are the real ones |

`_common/` modules have no README and are never called from a unit; a unit
always calls a public module.

## Variables and outputs

- Every variable has `type` and `description` (English, at most three
  lines; the reasoning goes into the README). `nullable` is written on every
  variable: `false`, or `true` where `null` is a value (`diagnostics`,
  `alerts`, an optional singleton).
- **`type = any` is not allowed.** `diagnostics` and `alerts` are typed
  objects; `null` switches them off.
- `validation` for values the API would reject late (enums, formats,
  `lock = CanNotDelete | ReadOnly | none`) and for every "exactly one of" the
  README states.
- A module takes the interfaces it implements, by the names of
  [azure.md](https://github.com/bearclave/governance/blob/main/standards/code/azure.md#interfaces-of-a-resource): `diagnostics`, `alerts`,
  `lock`, `private_endpoints`, `role_assignments` (principals as `group:`,
  `sp:`, `id:`).
- The collection variable comes first; the inputs that resolve it
  (`principal_ids`, `named_locations`) follow alphabetically.
- No deprecated provider attribute: the module moves to its replacement in
  the release that raises the lower bound.
- Every output has a `description`.

## Lookups and the baseline

The module resolves what a unit must not write: the IDs of built-in roles and
policy definitions by name, the client ID of an own application by display
name, groups by name (`entra/_common/principals`), the ABAC condition from role
and principal names. It enforces the security baseline a unit cannot switch
off. A value that differs between environments is an input the unit takes
from the context.

## Style

- `terraform fmt`; names `snake_case`; `this` for the single main resource,
  satellites by role (`lock`, `rbac`).
- Collections as `map(object)` keyed by a stable name, iterated with
  `for_each`; `count` only for a condition.
- Principals and groups are looked up only through
  `entra/_common/principals`: a field that takes only groups takes the bare
  name, a mixed field `group:`, `sp:`, `user:` (`user:` only in group
  membership); `id:` only for one the same change creates.
- `lifecycle.ignore_changes` only with a one-line comment at the lifecycle
  naming the process that owns the attribute, or the provider gap and version
  it works around (details in the README).
- Deletion protection is the `lock` input, with one line above the resource:
  `# tflint-ignore: azurerm_resources_missing_prevent_destroy # var.lock protects it, review YYYY-MM-DD`;
  `prevent_destroy` only where no lock can apply, with the reason in the
  README.
- Every `msgraph_*` resource and data source sets `api_version`; the API
  version of an azapi resource is a constant of the module
  (`locals { api = "..." }`), changed by a module release.
- One lower bound per provider in the repository; a module that needs a newer
  one says why in its README.
- A free-text field the platform shows on the object (`comment` on RouterOS)
  carries the object's key; no marker text.
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
`validation` or a `precondition` catches before the apply: a condition that
needs only variables is a `validation` (the break-glass exclusion of every
Conditional Access policy).

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
  gets a `moved` block in the module. The consumer's pull request changes
  `modules_ref` and the source paths together, with an empty plan.

## Checks

| Check | Holds |
|---|---|
| `terraform fmt -check`, `terraform validate` | per module |
| `tflint` (managed `.tflint.hcl`) | recommended preset, azurerm ruleset, `terraform_documented_variables`, `terraform_documented_outputs`, `terraform_typed_variables`, `terraform_standard_module_structure`, `terraform_naming_convention`, `terraform_unused_declarations` |
| `trivy config` | HIGH and CRITICAL fail; an accepted finding in `.github/trivyignore.yaml` with the reason and a review date |
| `terraform test` | every module with `tests/` |
| terraform-docs | the README block of every module is current (`--output-check`, version in `.github/tool-versions.env`) |
| catalog | `docs/catalog.md` is what `bearclave/ci` `src/terraform/catalog.py` writes |
| conventions | `variable-no-default`, `variable-documented`; planned: `variable-nullable`, `module-top-folder` (`modules/`) |

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
- `schema.json` describes the same object, key for key, and carries every
  `validation` JSON Schema can express, so that `pr / check` refuses what the
  plan would; a test passes the README example through the variable
  ([Files of a module](#files-of-a-module)).
- Where the provider has a list data source, a `check` compares the names in
  the tenant with the keys ([m365.md](https://github.com/bearclave/governance/blob/main/standards/code/m365.md#drift)).
- Over a generic provider (`msgraph_resource`, `azapi_resource`) the module
  is named after the API type (`conditional-access-policies`), never after
  the resource; it validates what the API would refuse late (`validation`).
- References to other objects come in by name or by key (`group:<name>`,
  the key of a named location) and are resolved in the module, by a data
  source or a dependency's output.

# Repository types

<!-- Standard of bearclave/governance (standards/repo-types.md). In other repositories this is a managed copy - change it in governance. -->

What a repository must have, by the custom property `pipeline` it calls.
On top of what every repository has ([repositories.md](https://github.com/bearclave/governance/blob/main/standards/repositories.md#what-every-repository-contains)).
A missing item is a `Debt` issue in that repository.

## IaC: `terragrunt`, `terragrunt-home`

`iac-azure`, `iac-github`, `iac-home` and the repositories of the Microsoft
cloud, one per security domain and apply identity:

| Repository | Manages | Providers |
|---|---|---|
| `iac-identity` | applications and service principals (with federated credentials, certificates, API permissions), directory and custom roles, groups and their members, enterprise and SAML applications, the Entra policies but Conditional Access | `azuread`, `msgraph` |
| `iac-conditional-access` | Conditional Access and what it uses: named locations, authentication strengths and contexts | `msgraph` |
| `iac-intune` | Intune and Defender for Endpoint: compliance, configuration, endpoint security, app protection, apps, update rings | `microsoft365wp`, `msgraph` |
| `iac-exchange` | Exchange Online and Defender for Office 365 | `exo` |
| `iac-purview` | sensitivity labels and their policies, DLP, retention, alert policies, the audit log | `scc`, `exo` |
| `iac-sentinel` | the content of Microsoft Sentinel: data connectors, analytic rules, hunting queries, parsers, watchlists, automation rules, workbooks, playbooks | `azurerm`, `azapi`, `azuread` (lookups) |

Teams, users, guests, licences and the organisation settings are not in
code ([code/m365.md](https://github.com/bearclave/governance/blob/main/standards/code/m365.md#in-code-and-out-of-code)). `iac-m365`
(Microsoft365DSC, pipeline `m365-dsc`) is retired: it hands its domains over
one by one ([code/terragrunt.md](https://github.com/bearclave/governance/blob/main/standards/code/terragrunt.md#handing-a-unit-over-to-another-repository))
and is archived after the last one; nothing new uses `m365-dsc`.

| Item | Rule |
|---|---|
| Pipeline | `ci.yml` from [templates/workflows/](https://github.com/bearclave/governance/tree/main/standards/templates/workflows/); plan on the pull request, apply from `main` ([pipelines.md](https://github.com/bearclave/governance/blob/main/standards/pipelines.md)); `subscription-id` empty where the repository manages no Azure resource |
| Drift | weekly, read-only identity, red on any difference (`drift.yml`, or the schedule in `ci.yml` for `terragrunt-home`); the Microsoft 365 repositories on different days, so that Graph and Exchange do not throttle them together |
| Environment | `<subject>-prod` (`azure-prod`, `github-prod`, `home-prod`, `identity-prod`, `conditional-access-prod`, `intune-prod`, `exchange-prod`, `purview-prod`, `sentinel-prod`), deployment branches `main` only, in `iac-github` |
| Identities | a plan and an apply identity ([identities.md](https://github.com/bearclave/governance/blob/main/standards/identities.md)), variables `AZURE_CLIENT_ID_PLAN` and `AZURE_CLIENT_ID_APPLY` |
| Versions | `.github/tool-versions.env` (Terraform, Terragrunt, trivy with sha256); providers exact in `account.hcl` ([toolchain.md](https://github.com/bearclave/governance/blob/main/standards/toolchain.md)) |
| Lock files | `.terraform.lock.hcl` committed in every unit, hashes `linux_amd64` and `darwin_arm64` |
| Layout | `root.hcl`, `account.hcl`, `_common/` (`units/`, `deps/`), `infrastructure/`; no `tools/`, no `deploy/` ([code/terragrunt.md](https://github.com/bearclave/governance/blob/main/standards/code/terragrunt.md)) |
| Collections | the Microsoft cloud repositories: collection units of `<key>.yaml`, the schema of each in `.github/schemas/`, `.vscode/settings.json` mapping them, one page per collection in `docs/` ([code/terragrunt.md](https://github.com/bearclave/governance/blob/main/standards/code/terragrunt.md#collections)) |
| Modules | from `iac-modules` at a tag (`modules_ref` in `account.hcl`), in `_common/units/<module>.hcl`, never a local path |
| Properties | `touches-production=true`; exact pin of `bearclave/ci` (`@vX.Y.Z`) |
| Docs | README Pipeline table with every trigger and identity; `docs/setup.md` (bootstrap as a procedure with its commands: state, identities, environment), `docs/manual-steps.md`, `docs/out-of-scope.md` where something is not in code; `docs/<collection>.md` per collection |
| State moves | `.github/state-moves.yaml` while a moved unit waits for its apply |

## Modules: `terraform-modules`

| Item | Rule |
|---|---|
| Pipeline | fmt, validate, tflint, trivy config, `terraform test`, terraform-docs up to date, conventions |
| Layout | `<platform>/<module>/` (`azure/key-vault/`), `<platform>/_common/`; `docs/catalog.md` generated; skeleton [templates/terraform-module/](https://github.com/bearclave/governance/tree/main/standards/templates/terraform-module/) |
| Rules | [code/terraform.md](https://github.com/bearclave/governance/blob/main/standards/code/terraform.md): no defaults, no `any`, provider lower bounds, a README with the terraform-docs block per public module, tests in `_common/` |
| Releases | one tag for the repository, `vX.Y.Z` ([versioning.md](https://github.com/bearclave/governance/blob/main/standards/versioning.md)); consumers pin it in `account.hcl` |
| Properties | `touches-production=true` while a consumer reads `main`; `false` once every consumer pins a tag |

## Application: `static-web-app`, `container`

| Item | Rule |
|---|---|
| Tests | in `tests/` (or the site's `tools/` checks for a static site), run in the pipeline on every pull request |
| Branches | default `staging`, production `prod`, moved only by `promote` ([workflow.md](https://github.com/bearclave/governance/blob/main/standards/workflow.md#applications-staging-and-prod)) |
| Environments | `staging` (deployment branch `staging`), `prod` (`prod`), `promote` (`staging`); a website also `preview` (`refs/pull/*/merge`) |
| Staging | a website: a Static Web App of its own (Free) in `rg-web-staging` of the production subscription, on its `*.azurestaticapps.net` name, with its own contact form Logic App; noindex added to the deployed config, never committed. A container: the staging container on its host |
| Deploy | a pull request: a preview (websites); `staging`: build once, deploy to staging; `prod`: the same build to production, no rebuild |
| Identity | a deploy identity per environment: production federated on `environment:prod`, staging on `environment:staging` (and `environment:preview`); a promote identity on `environment:promote` that reads only the key of the App `bearclave-promote` ([identities.md](https://github.com/bearclave/governance/blob/main/standards/identities.md)); an application on the NAS: the environment secret `DEPLOY_KEY` of each environment, an SSH key with a forced command ([security.md](https://github.com/bearclave/governance/blob/main/standards/security.md#secrets)) |
| Smoke test | after every deployment: the site or the service answers and serves the new version, noindex outside production only; a failure makes the run red. An application on the NAS (`app-home`) deploys from the NAS runner and is read through the router's reverse proxy; the image's `HEALTHCHECK` decides when the deployment is done |
| Artefact | built once per commit on `staging` and deployed unchanged to production: a website the artifact `site` of its staging run (kept 30 days, the artifact retention of the organisation; an older commit is not promoted), a container the image tagged with the commit and the version (`vX.Y.Z`), never only a branch name |
| Release | `promote` ([versioning.md](https://github.com/bearclave/governance/blob/main/standards/versioning.md#applications)): fast-forward of `prod`, then, once production is deployed, tag `vX.Y.Z` and release by `bearclave-promote` |
| Rollback | forward only: a `revert:` pull request into `staging`, then `promote`; `prod` never moves back. A failed production deployment of a good commit: a manual run of `ci` on `prod` (the same build again). Written in the README |
| Dependencies | `.github/dependabot.yml` for its ecosystems ([templates/dependabot.yml](https://github.com/bearclave/governance/blob/main/standards/templates/dependabot.yml)) |
| Properties | `touches-production=true`; exact pin of `bearclave/ci` |
| Docs | README: what runs where (staging, production), how to run it locally, how to ship to staging and promote, the smoke test, rollback |

A new kind of application (Azure Functions, a container on another host)
gets its shared workflow in `ci` and its row here before the first
deployment.

## Know-how, tools and content: `checks`, `content-sentinel`

`governance`, `handbook`, `ops-tools`, `.github`, `vs-code-extensions`,
`app-office-templates` (`checks`); `content-sentinel`, the catalogue of all
Sentinel content, deploys nothing. Once it is in the YAML format of
`iac-sentinel` it has no pipeline of its own: `pipeline` `checks` without a
command, and `pr / check` validates every file against the schemas
(`collection-schema`); until then `content-sentinel.yml` (Sentinel lint,
generated views up to date). `iac-sentinel` holds only what is deployed
([code/sentinel.md](https://github.com/bearclave/governance/blob/main/standards/code/sentinel.md#content-sentinel)).

| Item | Rule |
|---|---|
| Pipeline | `checks.yml`: security scan, ESLint where there is JavaScript, the repository's own command (tests, generators up to date); `content-sentinel.yml` until the catalogue is converted |
| Docs | Markdown rules of [documentation.md](https://github.com/bearclave/governance/blob/main/standards/documentation.md#markdown); links work |
| Releases | only where something is consumed by reference ([versioning.md](https://github.com/bearclave/governance/blob/main/standards/versioning.md)) |
| Properties | `touches-production=false`; `direct-commit=true` only where the owner writes straight to `main` |

## Shared workflows: `ci`

| Item | Rule |
|---|---|
| Pipeline | its own `ci.yml`: unit tests, self-tests of the checks, generated workflows up to date (`tools/build.py --check`) |
| Releases | `vX.Y.Z` and the major tag `vX` ([versioning.md](https://github.com/bearclave/governance/blob/main/standards/versioning.md)) |
| Identities | one bot identity per workflow purpose, each in its own environment |
| Docs | README per workflow; `docs/identities.md`, the inventory of every identity a workflow uses |
| Properties | `touches-production=true`: an apply pipeline runs its code |

## Templates: `template`

Copied to customers: nothing names the organisation or a tenant, local
copies of the shared workflows, a
version per handover ([versioning.md](https://github.com/bearclave/governance/blob/main/standards/versioning.md)).

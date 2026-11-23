---
paths:
  - ".github/**"
  - "**/*.hcl"
  - "**/*.tf"
---

# Identities

<!-- Standard of bearclave/governance (standards/identities.md). In other repositories this is a managed copy - change it in governance. -->

Which identity a workflow signs in with, what it may do and how it is
federated. Every identity a GitHub workflow uses is listed, with its
credentials and permissions, in the inventory
[`bearclave/ci` `docs/identities.md`](https://github.com/bearclave/ci/blob/main/docs/identities.md),
which links the page each repository keeps
([Documented in the repository](#documented-in-the-repository)); this page
holds the rules both follow.

## When a new identity is created

**One identity per pipeline and purpose.** A new identity is created when:

- a repository gets a pipeline that signs in to Azure, Entra, Microsoft 365
  or a GitHub App key (a new `iac-*` repository gets two: plan and apply);
- an existing workflow gains a purpose its identity does not have (`deps`
  and `ci-pins` got `sp-ci-deps` instead of reusing `sp-ci-major-tag`);
- one identity would otherwise serve a read-only and a writing job.

Never reused across repositories, never shared between a pull request and an
apply, never a person's account in a pipeline.

## Types

| Type | Used by | Runs on | May |
|---|---|---|---|
| **plan** | the plan of a pull request and the weekly drift | `pull_request`, schedule on `main` | read everything it plans; write nothing it manages |
| **apply** | the apply after a merge | the production environment only | write what the repository manages, nothing else |
| **bot** | workflows that act on GitHub as `bearclave-bot` (`major-tag`, `deps`, `ci-pins`, `project-sync`), or as `bearclave-promote` (`promote` of an application) | its own environment in `ci`; the environment `promote` of the application | read one App key: `bearclave-bot`'s in `kv-githubbot`, `bearclave-promote`'s in `kv-ghpromote` - never both |
| **deploy** | an application's deployment, one identity per environment | the environment `prod`, or `staging` (with `preview`), of the application | deploy its one resource (the staging identity: the staging resource only, and delete its previews) |

An application on the NAS (`app-home`) has no deploy identity: nothing in
Azure is deployed, and its deploy job signs in to the NAS over SSH with the
environment secret `DEPLOY_KEY` of each environment
([security.md](https://github.com/bearclave/governance/blob/main/standards/security.md#secrets)).

A bot identity is one per workflow purpose, even when they read the same
key: the credential decides which workflow can sign in, and a narrower
federation is a smaller blast radius.

## Permissions per type

| Permission | plan | apply | bot | deploy |
|---|---|---|---|---|
| Terraform state | Storage Blob Data Contributor on its own container only: the plan takes the state lock (a lease on the blob) and the first plan of a new unit creates the blob - never on the whole account | Storage Blob Data Contributor on its own container | - | - |
| Key Vault | Key Vault Secrets User on the read-only vault of its pipeline (`kv-githubro`); in `kv-home` only the single secrets a plan needs (`secret_role_assignments`), never a write credential | Key Vault Secrets User on the vault of its pipeline (`kv-github`, `kv-home`) | Key Vault Secrets User on `kv-githubbot`; a promote identity only on `kv-ghpromote` | - |
| Azure RBAC | Reader on the scope it plans | the narrowest role that applies (Contributor; where it assigns roles, Role Based Access Control Administrator with a condition that names the roles, never Owner) on the scope it manages | - | the narrowest role that deploys the resource (e.g. `Website Contributor` / Static Web App deployment, not Contributor on the resource group) |
| Entra roles, Graph | Global Reader, `*.Read.All` application permissions | the admin roles and `*.ReadWrite.*` permissions the code manages, each named in the identity's description | - | - |
| GitHub App | the read-only App (`bearclave-iac-github-read`): all permissions `read` but organisation and repository Administration `write` - GitHub refuses a read of an organisation ruleset and leaves the merge settings out of a repository without it (`iac-github` `docs/setup.md`) | the App that writes (`bearclave-iac-github`) | `bearclave-bot`: contents, pull requests, issues, workflows `write`, projects `write`, major-tag bypass; `bearclave-promote` (installed only on the applications): contents, workflows `write`, the bypass of `prod` and of `v*` tag creation there | - |

- No deploy token, client secret or certificate read by a plan identity.
  Known exception: `sp-tf-plan` holds `Static Web App Secret Reader` on both
  sites, because azurerm reads the deployment token on every refresh of
  `azurerm_static_web_app`. It goes once the module stops reading it
  (`api_key` ignored or the site through azapi).
- A Tier-0 permission (Owner or Role Based Access Control Administrator on
  a management group, Privileged Role Administrator,
  `RoleManagement.ReadWrite.Directory`, `Directory.ReadWrite.All`,
  `Application.ReadWrite.All`, `AppRoleAssignment.ReadWrite.All`)
  only on an **apply** identity, federated only on its environment. In Entra
  ID only `sp-tf-identity` holds Tier-0.
- Graph permissions are **application permissions**, never delegated; an
  Entra role only where the API knows no permission (Exchange, Purview,
  the Entra diagnostic setting).
- The description of the application says what it may do; the inventory
  repeats it.

## Federated credentials

| Type | Subject(s) | Never |
|---|---|---|
| plan | `<prefix>:pull_request` and `<prefix>:ref:refs/heads/main` (the drift runs on a schedule from `main`) | an environment |
| apply | `<prefix>:environment:<prod environment>` only (`<subject>-prod`, [repo-types.md](https://github.com/bearclave/governance/blob/main/standards/repo-types.md#iac-terragrunt-terragrunt-home)) | `ref:refs/heads/main`, `pull_request` |
| bot | `<prefix>:environment:<workflow environment>` of `ci` (`major-tag`, `deps`, `project-sync`), `<prefix>:environment:promote` of an application (recommended: a flexible credential that also requires `job_workflow_ref` of a released `ci` `promote.yml`, [`bearclave/ci` docs/applications.md](https://github.com/bearclave/ci/blob/main/docs/applications.md)) | a branch or `pull_request` |
| deploy | `<prefix>:environment:prod`; the staging identity `<prefix>:environment:staging` and `<prefix>:environment:preview` | a branch or `pull_request` |

- `<prefix>` is the immutable subject with the organisation and repository
  IDs: `repo:bearclave@328058774/<repository>@<repository id>`. A credential
  cannot pass to a re-created repository; a renamed repository needs a new
  credential under its new name (the old one is removed after the rename).
- An environment named in a subject has a deployment branch policy that
  allows `main` only ([`bearclave/iac-github`](https://github.com/bearclave/iac-github)),
  so only a run of `main` reaches the identity. In an application `prod`
  allows `prod`, `staging` and `promote` allow `staging`, and `preview`
  allows `refs/pull/*/merge` (the runs of pull requests; the identity
  reaches only the staging resource).
- The job that signs in names the environment (`environment:` in the shared
  workflow); a job without it gets a `ref` subject and is refused.

## Names

| What | Pattern | Example |
|---|---|---|
| plan | `sp-tf-<pipeline>-plan` (`sp-tf-plan` for `iac-azure`) | `sp-tf-home-plan` |
| apply | `sp-tf-<pipeline>` (`sp-tf-platform` for `iac-azure`) | `sp-tf-github` |
| bot | `sp-ci-<purpose>`, `sp-<purpose>` | `sp-ci-deps`, `sp-project-sync` |
| deploy | `sp-<kind>-<subject>-prod`, `sp-<kind>-<subject>-staging` | `sp-swa-sro-prod`, `sp-swa-sro-staging` |
| promote | `sp-ci-promote-<subject>` | `sp-ci-promote-web-sro` |
| federated credential | `gh-<application without sp->-<purpose>` | `gh-tf-home-plan` |
| GitHub variable | `AZURE_CLIENT_ID_PLAN`, `AZURE_CLIENT_ID_APPLY`, `AZURE_CLIENT_ID_<PURPOSE>` | `AZURE_CLIENT_ID_DEPS`; an application `AZURE_CLIENT_ID` (prod), `AZURE_CLIENT_ID_STAGING`, `AZURE_CLIENT_ID_PROMOTE` |

Descriptions are English and say what the identity may do and what it may
not.

## The repositories of the Microsoft cloud

Each repository has its own pair, `sp-tf-<subject>-plan` and
`sp-tf-<subject>` (`sp-tf-identity`, `sp-tf-conditional-access`,
`sp-tf-intune`, `sp-tf-exchange`, `sp-tf-purview`, `sp-tf-sentinel`), its
state container `tfstate-<subject>` (Storage Blob Data Contributor for both,
on that container only: every unit of a new repository is new, and its first
plan creates the blob) and nothing of another domain: a repository more is a
narrower identity, never the sum of two admin roles.
Microsoft Graph application permissions unless another API is named:

| Repository | plan | apply |
|---|---|---|
| `iac-identity` | `Application.Read.All`, `Directory.Read.All`, `RoleManagement.Read.Directory`, `Policy.Read.All`, `AccessReview.Read.All`; Key Vault Reader and Reader on the vault with the public certificates of the applications (the public part: never Key Vault Certificate User, which reads the private key) | `Application.ReadWrite.All`, `AppRoleAssignment.ReadWrite.All`, `DelegatedPermissionGrant.ReadWrite.All`, `RoleManagement.ReadWrite.Directory` (roles, role-assignable groups and their members), `Group.ReadWrite.All`, `GroupMember.ReadWrite.All`, `User.Read.All`, `Policy.Read.All`, `Policy.ReadWrite.ApplicationConfiguration`, `Policy.ReadWrite.B2BManagementPolicy`, `Policy.ReadWrite.ConsentRequest`, `Policy.ReadWrite.AuthenticationMethod`, `Policy.ReadWrite.Authorization`, `Policy.ReadWrite.CrossTenantAccess`, `Policy.ReadWrite.ExternalIdentities`, `Policy.ReadWrite.SecurityDefaults`, `Policy.ReadWrite.IdentityProtection`, `Policy.ReadWrite.DeviceConfiguration`, `AccessReview.ReadWrite.All`, `Directory.ReadWrite.All` (only the directory settings `/settings` need it); Key Vault Reader and Reader on the same vault |
| `iac-conditional-access` | `Policy.Read.All`, `Application.Read.All`, `Group.Read.All`, `RoleManagement.Read.Directory` | the same and `Policy.ReadWrite.ConditionalAccess` (policies, named locations, authentication strengths and contexts) |
| `iac-intune` | `DeviceManagementConfiguration.Read.All`, `DeviceManagementApps.Read.All`, `DeviceManagementServiceConfig.Read.All`, `DeviceManagementRBAC.Read.All`, `Group.Read.All` | `DeviceManagementConfiguration.ReadWrite.All`, `DeviceManagementApps.ReadWrite.All`, `DeviceManagementServiceConfig.ReadWrite.All`, `DeviceManagementRBAC.Read.All`, `Group.Read.All`; never `DeviceManagementManagedDevices.*`, `Device.ReadWrite.All` or Intune Administrator |
| `iac-exchange` | Office 365 Exchange Online `Exchange.ManageAsApp`, the Entra role Global Reader and the Exchange custom role View-Only Role Management (`Get-ServicePrincipal`, `Get-ManagementScope`, `Get-ManagementRoleAssignment`; made by hand, `docs/manual-steps.md`); `Application.Read.All` (`sp:` of RBAC for Applications), `Group.Read.All` (groups by name) | `Exchange.ManageAsApp` and the Entra role Exchange Administrator; `Application.Read.All`, `Group.Read.All` |
| `iac-purview` | `Exchange.ManageAsApp` and Global Reader; `Group.Read.All` only where a module looks groups up (rights of the labels) | `Exchange.ManageAsApp` and the Entra role Compliance Administrator (labels, label policies, DLP, retention, alert policies); the same `Group.Read.All` |
| `iac-sentinel` | Reader, Microsoft Sentinel Reader and Log Analytics Reader on the resource group of the workspace, Reader on the resource group of the playbooks; `Group.Read.All`, `User.Read.All`, `Application.Read.All` (watchlists made from groups, `sp:`); the Entra role Security Reader (the Entra diagnostic setting) | Microsoft Sentinel Contributor on the resource group of the workspace (rules, saved searches, workbooks, connectors) and Log Analytics Contributor there (the diagnostic setting `SentinelHealth` of the workspace), Logic App Contributor on the resource group of the playbooks, Role Based Access Control Administrator on both with a condition that allows only the Sentinel roles a playbook's identity needs; the Entra role Security Administrator (the Entra diagnostic setting, the Entra ID, Office 365 and Defender XDR connectors - a narrower role is checked on Learn before it is granted); nothing on subscriptions (Defender for Cloud comes through Defender XDR); the same Graph permissions as plan |

- **Never in these identities:** a permission of another domain, a
  `*.ReadWrite.All` on a plan identity, delegated permissions, a client
  secret. The Graph permissions a playbook's managed identity needs are
  granted in `iac-identity` (`sp:<Logic App>`), never by `iac-sentinel`.
- **Exchange and Purview** read the same `ARM_*` variables as the other
  providers and sign in with the federated token; that Global Reader is
  enough for the plan of Exchange and Security & Compliance app-only is
  checked in the read-only proof of concept before the first import. The
  unified audit log stays out of code: Compliance Administrator does not hold
  the Exchange role Audit Logs. A certificate, should
  a workload refuse the federated token, only for the apply identity, in the
  Key Vault of the pipeline.
- **The apply identity of `iac-sentinel` is watched**: any sign-in of
  `sp-tf-sentinel` that is not the pipeline raises an analytic rule of its
  own; the plan identity stays read-only.
- **A Logic App Standard** (prepared, not deployed) adds to `sp-tf-sentinel`
  what its first instance needs - the plan, the storage account, private
  endpoints - in the pull request that deploys it.

## Where they are declared

| What | Where |
|---|---|
| The application, its service principal, federated credentials, certificate and API permissions | `iac-identity`: one `<application>.yaml` in `infrastructure/applications/app-registrations/` |
| Entra roles | `iac-identity` `infrastructure/roles/directory-roles/` |
| Azure RBAC, Key Vault and state access | the unit of the resource, `role_assignments` |
| Client ID in GitHub | the repository's unit in `iac-github` names the application (`client_ids = { AZURE_CLIENT_ID_PLAN = "sp-tf-<subject>-plan" }`); `github/repository` resolves it by display name, so `sp-tf-github-plan` and `sp-tf-github` hold `Application.Read.All` |
| GitHub Apps | procedure in `iac-github` `docs/setup.md` (the key in the Key Vault); the permissions are the table above |
| Documentation | `docs/identities.md` of every repository whose pipeline signs in; the inventory in `bearclave/ci` `docs/identities.md` links them |

- **Removing an identity:** before the pull request, `git grep` its name and
  client ID in every clone (`sp:<name>` in `_common/*.hcl`, YAML, docs) and the
  owner's wiki; each reference goes in the same or an earlier pull request.
- **App permissions:** a pull request that adds a resource type or an API call
  checks the permission GitHub requires for it and updates the table in
  `iac-github` `docs/setup.md`. A 403 or 422 in an apply is a missing
  permission, not a reason to rerun. Instructions for the owner name the exact
  page and section (Organization or Repository permissions).

A new repository's identities, in this order: `iac-github` creates the
repository and its environment; `iac-identity` the two applications, their
federated credentials (the repository ID is a literal, it never changes)
and permissions; `iac-azure` the state container and its role assignments;
`iac-github` the variables. The inventory row comes with the workflow change
in `ci` or with the first pull request that uses it.

A repository that refers to a principal by name (`sp:<name>`) is merged only
after the pull request that creates it has applied; otherwise the reference is
a `dependency` output.

## Documented in the repository

Every repository whose pipeline signs in has `docs/identities.md`, in one
format. One table, one row per permission:

| Identity | Used by | Role | Scope | Why |
|---|---|---|---|---|
| `sp-tf-exchange-plan` | plan, drift | Entra role Global Reader | tenant | reads what Exchange plans |
| `sp-tf-exchange` | apply (`exchange-prod`) | Storage Blob Data Contributor | container `tfstate-exchange` | writes its state |

- **Identity** is the display name; **used by** the workflow and
  environment (or the device account); **role** is one permission - an Azure
  role, an Entra role, a Graph application permission, an API role, a Key
  Vault role - with the API named where it is not Graph; **scope** is where it
  applies; **why** is the resource type or API that needs it. A GitHub App
  and a device account the pipeline uses (RouterOS `tf-plan`, `tf-apply`) are
  rows too. The file that declares each is named under the table or in a
  `Declared in` column.
- Below the table `## From scratch`: what each apply identity cannot create
  by itself and the manual step that does (`docs/manual-steps.md`).
- **Every deploy identity holds the permissions to redo the whole deployment
  from scratch**; what it cannot get (a Tier-0 grant, a custom Exchange role)
  is in `## From scratch`, never an unwritten step.
- The table is what a deploy from scratch needs; the inventory in `ci` links
  it instead of repeating it.

# Repository layout

<!-- Standard of bearclave/governance (standards/layout.md). In other repositories this is a managed copy - change it in governance. -->

Every repository is laid out the same way, so a person or an agent finds
things without reading the README first. The machine-readable form is
[layout.yaml](https://github.com/bearclave/governance/blob/main/standards/layout.yaml); the layout check in `pr / check` refuses a pull
request whose root breaks it.

## The root is minimal

Only what something reads from the root by itself:

| What | Files |
|---|---|
| Git and GitHub | `README.md`, `.gitignore`, `.gitattributes`, `.editorconfig`, `LICENSE` (only where the code is handed over) |
| Ecosystem manifests | `package.json` + lock, `pyproject.toml` + `uv.lock`, `Dockerfile`, `.dockerignore` |
| Terragrunt context | `root.hcl`, `account.hcl` (terragrunt pipelines) |
| Tools that look in the root | `.gitleaks.toml`, `.tflint.hcl`, `eslint.config.mjs`, `.mcp.json` (Claude Code) |
| Folders | the vocabulary below and the content folders of the pipeline |

Everything else goes into a folder. A config that only the pipeline reads
belongs in `.github/`, a script in `tools/` (not in a Terragrunt
repository), a document in `docs/`.

## `.claude/` and `.github/`

| Path | Holds |
|---|---|
| `.claude/CLAUDE.md` | working rules specific to the repository ([templates/CLAUDE.md](https://github.com/bearclave/governance/blob/main/standards/templates/CLAUDE.md), at most 30 lines) |
| `.claude/rules/`, `.claude/settings.json` | managed copies (`tools/sync.py`): the always-rules and the language rules of the languages the repository has |
| `.github/workflows/` | callers of `bearclave/ci` |
| `.github/CODEOWNERS`, `.github/SECURITY.md`, `.github/ISSUE_TEMPLATE/`, `.github/pull_request_template.md` | managed copies (`tools/sync.py`) |
| `.github/markdownlint.yaml` | markdownlint rules that differ from `ci` |
| `.github/dependabot.yml`, `.github/release.yml`, `.github/tool-versions.env` | GitHub and CI |
| `.github/conventions.yaml` | conventions check, and the exceptions to this layout |
| `.github/state-moves.yaml` | Terragrunt: the units whose state the owner moved, until the apply that used them ([code/terragrunt.md](https://github.com/bearclave/governance/blob/main/standards/code/terragrunt.md#moving-a-unit)) |
| `.github/schemas/` | `<module>.schema.json`: the JSON Schema of each collection, copied from its module ([code/terragrunt.md](https://github.com/bearclave/governance/blob/main/standards/code/terragrunt.md#schemas)) |
| `.github/sentinel-lint.yaml` | Sentinel lint |
| `.github/trivyignore.yaml` | accepted trivy findings, each with a reason and a review date |
| `.github/pssa.psd1` | PSScriptAnalyzer settings |

No scripts in `.github/`. The exception are the `template-*` repositories,
whose exported pipeline copies carry their scripts in `.github/scripts/`.

## Folders

English, plural, the same names everywhere:

| Folder | Holds |
|---|---|
| `src/` | code |
| `tests/` | tests, mirroring `src/` |
| `tools/` | scripts a person runs for this repository: build, generators, bootstrap, operations |
| `docs/` | longer documents (below) |
| `_common/` | shared code that is neither a unit nor a module (Terragrunt includes, `_common/` of modules) |
| `.vscode/` | `settings.json` only, with what the editor needs to read the repository: the `yaml.schemas` of the collections. No extensions list, no personal settings |

Only in the group `app`, each with a line in the README Layout section
(`content/` also in the group `template`, for a catalogue):

| Folder | Holds |
|---|---|
| `config/` | configuration the code reads |
| `content/` | content the product publishes or renders |
| `data/` | input data |
| `deploy/` | what runs the product where it is deployed |
| `lib/` | libraries the code shares |

Content folders of a pipeline:

| Pipeline | Content folders |
|---|---|
| `terragrunt`, `terragrunt-home` | `infrastructure/` (required), every unit below it |
| `terraform-modules` | `modules/`, one folder per platform below it (`modules/azure/`, `modules/entra/`, ...) |
| `static-web-app` | `api/` (the Functions folder of the site) |
| `content-sentinel` | `content/` (group `template`) and the pipeline `checks` ([code/sentinel.md](https://github.com/bearclave/governance/blob/main/standards/code/sentinel.md#content-sentinel)) |
| `ci` | `templates/` |

A script belongs in `tools/` only when it serves this repository. Logic
that the pipelines of several repositories run lives in `bearclave/ci`
(`src/`, carried by the shared workflow); a script used across repositories
or against a customer's tenant lives in the know-how repository; a step of
one pipeline is not a script of its own.

A Terragrunt repository (`terragrunt`, `terragrunt-home`) has **no `tools/`
and no `deploy/`** (`forbidden` in `layout.yaml`):

| Script | Where |
|---|---|
| run by the pipeline | `bearclave/ci` (`self-hosted-tools.yml`, `home-vpn.sh` of `terragrunt-home`) |
| run by a person (new peer, new secret, an App) | the know-how repository, `tools/<repository>/` |
| bootstrap | a procedure with its commands in `docs/setup.md`, never code |
| files deployed outside Terraform | the repository of what they deploy (`app-nas`) |

A script moves to the know-how repository in a pull request merged **before**
the one that deletes it here, and the deleting pull request links it (the
scripts of `iac-home` were deleted and never arrived).

A Terragrunt unit directory moves only with its state
([code/terragrunt.md](https://github.com/bearclave/governance/blob/main/standards/code/terragrunt.md#moving-a-unit)).

Names of product content that is Czech by nature (the templates of
`app-office-templates`) may stay Czech; the folders around them follow the
vocabulary.

## docs/

- Flat, kebab-case names: `docs/break-glass.md`.
- Fixed names where they apply: `setup.md`, `identities.md`,
  `manual-steps.md`, `out-of-scope.md`, `decisions.md`, `catalog.md`.
- File names are English ([README.md](https://github.com/bearclave/governance/blob/main/standards/README.md#scope-and-language)).
- A subfolder only once `docs/` has eight files or more.
- The README links every file in `docs/`.

## Exceptions

An entry the layout does not allow is either moved or listed with a reason
in `.github/conventions.yaml` (the conventions check ignores this section):

```yaml
layout:
  exceptions:
    - path: standards
      reason: The standards are the content of this repository.
```

`path` is a root entry, or `.github/<name>`, `docs/<name>`; globs are
allowed. An exception without a reason fails the check.

## .gitattributes

Managed by `tools/sync.py` in every repository: `* text=auto eol=lf`, the
binary formats, and below the marker a section the repository owns - the
paths it generates, marked `linguist-generated=true` so that they collapse
in diffs and do not count for the size label.

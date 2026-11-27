---
paths:
  - "**/*.md"
---

# Documentation

<!-- Standard of bearclave/governance (standards/documentation.md). In other repositories this is a managed copy - change it in governance. -->

## What goes where

| Document | Holds | Does not hold |
|---|---|---|
| `README.md` | what the repository is, how it fits, how to change it, its pipeline | working rules for agents, history of changes |
| `.claude/CLAUDE.md` | working rules specific to this repository | documentation, organisation standards (those are in `.claude/rules/`) |
| `docs/*.md` | longer topics of the repository: `setup.md`, `identities.md`, `manual-steps.md`, `out-of-scope.md`, `decisions.md`, `catalog.md` ([layout.md](https://github.com/bearclave/governance/blob/main/standards/layout.md#docs)) | anything the README already says |
| `README.md` of a module | its inputs, outputs and why it is built the way it is | which units use it, their values, the rules of a live repository |
| `bearclave/knowhow` | know-how across repositories, `gotchas/` | documentation of one repository |
| LLM wiki | the owner's processes and where things are | technical documentation - it links here |

One fact lives in one place; everywhere else links to it.

## README structure

From [templates/README.md](https://github.com/bearclave/governance/blob/main/standards/templates/README.md). Sections in this order; a
section that does not apply is left out, not left empty.

1. **`# <repository>`** and one sentence: what it manages and for whom.
2. **How it fits** - what it depends on and what depends on it, a small
   diagram if more than two repositories are involved.
3. **Layout** - the directory tree to the depth that explains the code. A
   repository whose content is documents (`knowhow`) indexes them here,
   one subsection per folder.
4. **How to make a change** - the commands to run locally (format, validate,
   plan, tests), and what a pull request gets checked for.
5. **Pipeline** - a table: trigger, what runs, identity. Which shared
   workflow of `bearclave/ci` it calls.
6. **Conventions** - only what is specific to this repository; the
   organisation rules are linked, not repeated.
7. **Gotchas** - links to `knowhow/gotchas/` entries that bite here.
8. **Repository settings** - the last section, the one line of the template
   [templates/README.md](https://github.com/bearclave/governance/blob/main/standards/templates/README.md#repository-settings), copied
   word for word with the repository name filled in; no check compares it,
   the reviewer does:
   > Branch protection, required checks, merge settings and labels are
   > defined in [`bearclave/iac-github`](https://github.com/bearclave/iac-github)
   > (`infrastructure/repositories/<repository>`). Engineering standards:
   > [`bearclave/governance`](https://github.com/bearclave/governance/tree/main/standards).

No other `##` sections: a repository-specific topic is a file in `docs/`,
linked from Layout. A fork of a public project (`terraform-provider-scc`)
keeps the upstream README and adds only the Repository settings section.

## Writing

- English, plain and short. Present tense, active voice: "The pipeline plans
  the units the change touches", not "Units will be planned".
- Explain **why** whenever a choice is not obvious; the code already says
  what.
- Tables for anything with more than two attributes; bullet lists over
  paragraphs; a heading every screenful.
- Code blocks with a language (` ```bash `, ` ```hcl `), commands that can be
  copied and run as they are, placeholders in `<angle brackets>`.
- Relative links inside the repository, full GitHub links to another one.
- No dates of "last updated", no author names, no changelog: git has them.
- Lines wrapped at about 80 characters in prose; tables and links may be
  longer.
- A document that describes behaviour is updated in the same pull request
  that changes the behaviour.
- A statement about a live setting (`stays off`, `installed on all
  repositories`) is checked with `gh api` or the portal in the pull request
  that writes it; a value that changes is not written (it is in Notion), a
  check date never.
- A mistake or trap found in practice is a `knowhow/gotchas/` entry, linked
  from the README Gotchas section.

## Markdown

| Rule | Limit |
|---|---|
| `README.md` | at most ~300 lines; anything longer moves to `docs/` and is linked |
| `.claude/CLAUDE.md` | at most 30 lines (`tools/sync.py` warns) |
| A file in `docs/` | one topic; split it past ~400 lines |
| Headings | one `#` per file (the title), then `##` and `###`; no level skipped; sentence case; no trailing punctuation; unique among siblings |
| Lists | `-` for bullets, `1.` for steps |
| Code blocks | fenced, with a language (none = plain text) |
| Links | relative inside the repository, full URL to another; every link resolves |
| HTML | only `<details>`, `<summary>`, `<br>`, anchors and comments (review; the lint allows placeholders in `<angle brackets>`) |

**Enforced** by `pr / check` of `bearclave/ci` in every repository:

- **`markdownlint-cli2`** with the rules of `ci` `src/markdownlint.yaml`:
  headings, lists, code fences, blank lines. A repository that needs another
  setting writes only that rule into its own `.github/markdownlint.yaml`
  (it replaces the rule, the rest stays), with a comment why; one place gets
  `<!-- markdownlint-disable-next-line MDxxx -->` and a reason.
- **`markdown-links`**: relative links and anchors in `*.md` resolve to a
  file and a heading in the repository.

`.claude/` is not linted: the copies there are linted in `governance`.

## Comments in code

The code stays as clean as possible; what it needs to be understood goes into
the README or `docs/`, not into comments.

- English only.
- Only when necessary: a reason that is not obvious, a constraint, a trap, a
  workaround - which names what it works around and links its issue or
  gotcha, so it can go once that is fixed.
- **One line each**, in `terragrunt.hcl`, context and topic files too; at
  most two elsewhere. No header comment: the path says what a file is (a
  shell script, a workflow caller, a compose file: one line of what it does).
  Triggers, design, reasons, usage and how it fits belong in the README (or
  `docs/`).
- Never restate the code below.
- No section banners and no decorative comments.
- No commented-out code; git keeps the old version.
- No `TODO` without an issue: `# TODO(#42): drop once the provider supports X`.

The language rules in `code/` follow this and do not repeat it.

| Rule | Enforced by |
|---|---|
| `TODO`/`FIXME` carries an issue number (`TODO(#42)`) | the check **`todo-issue`** of the conventions in `bearclave/ci` |
| no commented-out code, English only, no banners, never restating the code | review (`/code-review`); no linter tells these apart reliably |

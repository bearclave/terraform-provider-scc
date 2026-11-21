# Issues and labels

<!-- Standard of bearclave/governance (standards/issues.md). In other repositories this is a managed copy - change it in governance. -->

GitHub issues are the **backlog of the code**: what is wrong in it and what
it should do. They are not tasks. Personal and company tasks, deadlines and
time stay in Notion; a Notion task may link an issue.

## Issue types

Set on the organisation (`bearclave/iac-github`), the same in every repository.

| Type | When | Example title |
|---|---|---|
| `Bug` | the code or the deployed state does something it should not, or drifts | `Drift in conditional access policy CA05` |
| `Feature` | new behaviour, resource, setting or page | `Block legacy authentication for all users` |
| `Debt` | works, but is wrong in a way that will cost later: a review finding left unfixed, a manual change not yet in code, a missing test, a workaround | `Move the tenant-wide audit retention into code` |
| `Security` | a weakness or hardening finding, in the code or in the tenant it manages | `Static Web App allows TLS 1.0` |

The GitHub type `Task` is disabled: tasks belong in Notion.

## Writing an issue

- **One problem per issue.** Larger work is a parent issue with sub-issues,
  one per pull request.
- **Title**: English, specific, no type prefix (the type is a field), no full
  stop. A noun phrase or an imperative: `Plan fails for a unit with no
  resources`, `Add Defender for Storage to sub-sec-prod`.
- **Body** from the issue form of its type ([templates/ISSUE_TEMPLATE/](https://github.com/bearclave/governance/tree/main/standards/templates/ISSUE_TEMPLATE/)):

| Type | Sections |
|---|---|
| Bug | What happens · What should happen · Where · Impact · How to reproduce · Evidence |
| Feature | Why · Proposal · Done when · Risks and impact |
| Debt | What is wrong · Why it matters · Proposed fix · Found in |
| Security | Finding · Exposure · Proposed fix · References |

- **Assignee**: the owner (organisation variable `OWNER`, today
  `matej-hrabalek`; the issue forms assign him), until the repository has
  another maintainer.
- **Priority** by label: `prio:high` (fix this week, or production is
  affected), no label (normal), `prio:low` (when convenient). A `Security`
  issue with real exposure is always `prio:high`.
- **Source** label when the issue did not come from a person's idea:
  `source:review`, `source:drift`, `source:manual-change`. A red drift run
  opens no issue by itself: whoever reads it (the owner, the morning
  briefing, an agent) opens a `Bug` with `source:drift` and the run link.
  The pipeline opens one by itself only when the verification after an
  apply fails ([pipelines.md](https://github.com/bearclave/governance/blob/main/standards/pipelines.md#rules-for-every-workflow)).
- **Closed** by the pull request that resolves it (`Closes #12` in its
  description), or by hand with a comment saying why it is not done
  (`not planned` / `duplicate`).

## When an agent creates an issue

An AI coding agent creates an issue - without asking - when it:

| Situation | Type | Labels |
|---|---|---|
| leaves a review finding unfixed in its pull request | `Debt` (or `Security`) | `source:review` |
| finds a bug outside the scope of its current task | `Bug` | |
| learns about a change made by hand in a tenant or portal that belongs in code | `Debt` | `source:manual-change` |
| sees a plan propose a change nobody made in code (drift) | `Bug` | `source:drift` |
| finds a security weakness | `Security` | `prio:high` when exposed |

It does **not** create an issue for the work of its current pull request, and
not for a question to the owner (that goes into the conversation).

Procedure:

1. Look for a duplicate first:
   `gh issue list --repo bearclave/<repo> --state all --search "<keywords>"`.
   A match gets a comment with the new information instead.
2. Write the body to a file from the section list above, English, with links
   to the pull request, run or file and line.
3. Create it with type, assignee and labels in one call:

   ```bash
   gh api -X POST repos/bearclave/<repo>/issues \
     -f title='<title>' -F body=@/tmp/issue.md -f type='<Bug|Feature|Debt|Security>' \
     -f 'assignees[]=<owner>' -f 'labels[]=source:review'
   ```

4. Name the issue (`#N` and link) in the pull request's Review section or in
   the reply to the owner.

## Labels

One set for every repository, defined in `bearclave/iac-github`. Issues and
pull requests share it; a label marked "PR" is set on pull requests only.

| Label | On | Meaning | Set by |
|---|---|---|---|
| `type:feat` `type:fix` `type:perf` `type:refactor` `type:docs` `type:test` `type:ci` `type:chore` `type:deps` `type:revert` | PR | the commit type of the title | `pr` check |
| `breaking` | PR | the title has `!`: consumers have to change | `pr` check |
| `size:xs` `size:s` `size:m` `size:l` `size:xl` | PR | changed lines, see [workflow.md](https://github.com/bearclave/governance/blob/main/standards/workflow.md#pull-requests) | `pr` check |
| `prio:high` | both | this week, or production is affected | person, agent |
| `prio:low` | both | when convenient | person, agent |
| `source:review` | issue | a review finding left unfixed | agent |
| `source:drift` | issue | found by a drift run | agent, drift run |
| `source:manual-change` | issue | a change made by hand that belongs in code | person, agent |
| `status:blocked` | both | waits on something outside the repository; a comment says on what | person, agent |
| `status:needs-decision` | both | waits on the owner's decision | person, agent |

No other labels: `iac-github` deletes every label not in this list, so a
repository-specific label (`area:*`) is not used; the scope of the title and
the path in the issue say the area.

Issues carry no `type:*` label: their type is the issue type. GitHub's
default labels (`bug`, `enhancement`, `good first issue`, ...) and
Dependabot's `dependencies`/`github_actions` are not used.

## Issues and pull requests

- A pull request does not need an issue. It needs one when it resolves a
  reported problem or a planned feature: `Closes #12` in Links closes the
  issue on merge and links both in the Project.
- Work larger than one pull request is a parent issue with a sub-issue per
  pull request; each pull request closes its sub-issue.
- An issue is never closed by a pull request that does not resolve it
  (`Refs #12` for a related one).

## Where issues show up

- Per repository on GitHub, and across the organisation in the Projects
  `IaC`, `Apps`, `Templates & content` and `Platform` (one per repository
  group): fields, views and what Status means are in
  [projects.md](https://github.com/bearclave/governance/blob/main/standards/projects.md).
- Every open issue and pull request is added to the Project of its
  repository's `group` by the daily workflow `project-sync` in `ci`
  (06:15 UTC; new items appear within a day). Nothing is set up per
  repository, and the Projects' own "Auto-add" workflow is off.
- In VS Code in the GitHub Overview view (assigned to me by type,
  `prio:high`, by repository).

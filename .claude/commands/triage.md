Adopt the triage persona and conduct an interactive intake interview to produce one or more GitHub issues with correct routing labels.

**Interactive mode**: This skill MUST pause and ask questions. Never create issues without explicit human confirmation of the draft. The triage agent does not commit, does not write files, and does not modify any repository content.

---

## Step 1 — Adopt the Persona

Read `CLAUDE.md` and confirm:
- The Triage Agent Rules section (routing table, phase determination)
- The GitHub Issue Routing section (required label set)

Then do a baseline source read before the interview begins:
- Run `find . -type f -name "*.go" | head -40` (or equivalent) to orient yourself to the file layout
- Read key entry points: `cmd/ldc-demo/commands/root.go`, `internal/state/`, `internal/tofu/`, `internal/kubeconfig/`

Greet the human:

> "I'm the triage agent. I'll ask a few questions to understand the issue, then draft and create the right GitHub issue(s). To start — can you describe the problem in one or two sentences?"

---

## Step 2 — Intake Interview

After the human's initial description, read the relevant source files to understand context before asking follow-up questions. Never ask the human what is in the code — look it up yourself.

Ask only the questions not already answered. Work through this list in order, skipping any already clear from context:

**Q1 — What did you observe?**
> "What exactly happened? (error message, unexpected behavior, missing feature, documentation gap, CI failure, etc.)"

**Q2 — Where in the system?**
> "Which part of the system was involved? For example: the state manager, tofu provisioner, kubeconfig retrieval, CLI command, CI workflow, documentation, or something else?"
>
> Once the human identifies an area, **read the relevant source files** to verify the code exists and understand what it does — do not ask the human to describe it.

**Q3 — How was it triggered?**
> "How did you encounter this? (running `make test`, cluster deployment, code review, reading docs, CI run, manual testing, etc.)"

**Q4 — Regression or new area?**
> "Was this working before, or is it a new area that has never worked?"

**Q5 — Logs or reproduction steps?**
> "Do you have logs, stack traces, or steps to reproduce? Paste or summarize them here."

Do not proceed to Step 3 until you have enough information to fill the routing table and draft the issue body.

---

## Step 3 — Categorize and Route

Before routing, **always perform a security scan** of the affected source files regardless of whether the human mentioned a security concern:
- Check for hardcoded credentials, secrets, or tokens
- Check for overly broad permissions or missing auth checks
- Check for secrets logged or exposed in error messages
- Check for unvalidated external input reaching sensitive operations

If any security concern is found, always add a `type/security` issue to the tuple list, even if the primary report is a bug or task.

Apply the Triage Routing Table from CLAUDE.md:

**Determine type(s):**
- `type/bug` — something broken that was working (or never worked as designed)
- `type/task` — work to be done (new coverage, new docs, new feature)
- `type/security` — credentials, secrets, or access control finding

**Determine persona(s):**

| Symptom | Assign to |
|---|---|
| Crash / panic / wrong CLI behavior / provider error | `persona/developer` + `type/bug` |
| Missing test coverage revealed by above | `persona/test-engineer` + `type/task` |
| Confusing docs / missing runbook entry / unclear README | `persona/docs` + `type/task` |
| Secret in logs / credential exposure / overpermission | `persona/security` + `type/security` |
| New feature design / architecture decision | `persona/product-designer` + `type/task` (product-designer will create downstream tasks) |
| CI failure / workflow error / Makefile / OpenTofu module bug | `persona/gitops-manager` + `type/bug` |
| E2E or acceptance test failure | `persona/qa` + `type/bug` |

**Determine phase** by the affected component:

| Component | Phase |
|---|---|
| `go.mod`, `Makefile`, `.github/workflows/**`, `.gitignore`, `.env.template`, CI pipeline | `phase/1-foundation` |
| `internal/state/**`, `internal/tofu/**`, `internal/kubeconfig/**`, `cmd/ldc-demo/commands/root.go`, `cmd/ldc-demo/commands/list.go` | `phase/2-core-logic` |
| `cmd/ldc-demo/commands/create.go`, `cmd/ldc-demo/commands/remove.go`, `cmd/ldc-demo/commands/get_kubeconfig.go`, `tofu/modules/**`, `configs/**` | `phase/3-integration` |
| `test/e2e/**`, `docs/quickstart.md`, `docs/runbook.md`, `docs/acceptance-criteria.md`, release pipeline | `phase/4-hardening` |

If the human cannot identify the file area, search the source code for relevant symbols, function names, or error strings from the human's description to determine the affected file and phase — do not ask the human which file is involved.

Build a list of `(persona, phase, type)` tuples — one per issue to create.

---

## Step 4 — Draft Issues

For each tuple, compose a complete draft using the appropriate template:

**`type/bug` template:**
```
Title: bug(<scope>): <one-line description>

Labels: persona/<name>, phase/<n>-<phase-name>, type/bug

## Description
<What is broken and how it was discovered>

## Steps to Reproduce
1. <step>

## Expected Behavior
<What should happen>

## Actual Behavior
<What actually happens, with any error output>

## Found By
<How the reporter encountered this>
```

**`type/task` template:**
```
Title: [<persona>] <one-line description>

Labels: persona/<name>, phase/<n>-<phase-name>, type/task

## Description
<What needs to be done and why>

## Acceptance Criteria
- [ ] <criterion>

## Files / Packages Affected
<Key files>

## Depends On
<Issue numbers, or "none">
```

**`type/security` template:**
```
Title: security(<scope>): <one-line description>

Labels: persona/security, phase/<n>-<phase-name>, type/security

## Finding
<What the security concern is>

## Risk
<What could go wrong>

## Evidence / Reproduction
<How to verify>

## Recommendation
<What needs to change>
```

Print all drafts separated by `---`.

---

## Step 5 — Confirm With Human

After printing all drafts, ask:

> "I'm ready to create the above issue(s). Please review:
> - Are the labels and routing correct?
> - Is the description accurate?
> - Any wording changes?
>
> Reply **'create'** to proceed, **'edit'** with your changes to revise, or **'cancel'** to abort."

- **'create'** → proceed to Step 6
- **'edit \<changes\>'** → apply changes, reprint updated draft(s), return to top of Step 5
- **'cancel'** → print `Triage cancelled. No issues were created.` and stop

Do not create any issues until the human explicitly confirms.

---

## Step 6 — Create Issues

For each confirmed issue, run:

```bash
gh issue create \
  --title "<title>" \
  --body "<body>" \
  --label "persona/<name>" \
  --label "phase/<n>-<phase-name>" \
  --label "type/<bug|task|security>"
```

**Label names must exactly match existing labels:**
- `persona/developer`, `persona/test-engineer`, `persona/security`, `persona/qa`, `persona/gitops-manager`, `persona/docs`, `persona/product-designer`
- `phase/1-foundation`, `phase/2-core-logic`, `phase/3-integration`, `phase/4-hardening`
- `type/bug`, `type/task`, `type/security`

**Multi-issue dependency:** When creating a follow-on issue (e.g., `test-engineer` task that depends on a `developer` bug fix), capture the issue number returned by the first `gh issue create` and fill it into the `Depends On` field of the second issue before creating it.

---

## Step 7 — Report

Print a summary:

```
Triage complete. Created <N> issue(s):

- #<number>: <title> [<labels>] — <url>
- #<number>: <title> [<labels>] — <url>

Routing:
- persona/<name> will address: <brief description>
- persona/<name> will address: <brief description>

To work these issues: /watch-work <persona-name>
```

Stop after printing the report.

# /persona-setup

Initializes a persona session for the ldc-demo project. Run this at the start of every persona session to verify your environment, review your scope, and either pick up an open issue or create a new feature worktree.

## Step 1 — Identify your persona

Ask the user which persona they are invoking, or infer it from the current working directory:

| Current directory ends with... | Persona |
|---|---|
| `losant-device-demo` (main clone) | merge-manager, triage, or developer/test-engineer setup |
| `losant-device-demo-worktrees/security` | security |
| `losant-device-demo-worktrees/gitops-manager` | gitops-manager |
| `losant-device-demo-worktrees/docs` | docs |
| `losant-device-demo-worktrees/qa` | qa |
| `losant-device-demo-worktrees/product-designer` | product-designer |
| `losant-device-demo-worktrees/developer-*` | developer |

If the directory is a `developer-*` worktree, prompt the test-engineer to use the same directory.

## Step 2 — Verify environment

Run the following checks and report any failures before proceeding:

```bash
# Confirm git branch matches expected persona branch
git branch --show-current

# Confirm working tree is clean (no unexpected staged changes)
git status

# Confirm .claude/commands/ is present (slash commands available)
ls .claude/commands/

# For developer worktrees: confirm go builds
go build ./...
```

If the branch does not match the persona, stop and instruct the user to start Claude from the correct worktree directory.

## Step 3 — Review persona scope

Remind the persona of their file ownership rules from CLAUDE.md:

- **security**: `.gitignore`, `.env.template`, CI security steps
- **gitops-manager**: `tofu/**`, `Makefile`, `.github/workflows/**`, `go.mod`, `go.sum`
- **docs**: `docs/**`, `README.md`, `CLAUDE.md`
- **qa**: `test/e2e/**`, `docs/acceptance-criteria.md`, `docs/runbook.md`
- **developer**: `cmd/**`, `internal/**`, `configs/**`
- **test-engineer**: `*_test.go`, `test/**`, mock implementations

Violations are caught by the merge-manager before any PR merges.

## Step 4 — Find open issues for this persona

Fetch open GitHub issues assigned to this persona's label:

```bash
gh issue list --repo mak3r/losant-device-demo --label "persona/<name>" --state open
```

Replace `<name>` with the persona name (e.g. `persona/developer`, `persona/security`).

Present the list to the user and ask which issue to work on, or suggest the lowest-numbered unblocked issue.

## Step 5 — Set up the worktree (developer and test-engineer only)

**For all other personas:** worktrees already exist — skip to Step 6.

**For developer:** after picking an issue, create a feature worktree from the main clone:

```bash
# Run from losant-device-demo/ (main clone)
FEATURE=<short-descriptor-matching-issue>
git worktree add ../losant-device-demo-worktrees/developer-$FEATURE -b feature/developer/$FEATURE

# Confirm
git worktree list
```

Then instruct the user:
> "Your worktree is ready at `../losant-device-demo-worktrees/developer-<feature>`. Open a new terminal, cd into that directory, and start a new Claude session there to begin implementation. The test-engineer should start their Claude session from the same directory."

**Do not begin implementation in the current (main clone) session.** The main clone stays on `main`.

## Step 6 — Confirm readiness and begin work

Summarize:
- Persona name and branch
- Working directory
- Issue being worked on (title, number, URL)
- Files in scope for this persona
- Any blockers (blocked-by issues that must close first)

Then either:
- Begin working on the issue if in a valid persona worktree
- Wait for the user to start a new session in the feature worktree (developer/test-engineer case)

## Handoff reminder

When work is complete, follow the handoff protocol in CLAUDE.md before closing any issue:
1. Commit and push the branch
2. Open a PR targeting `develop` (or `main` if no develop branch)
3. Re-label or create a new issue for the next persona in the chain
4. Comment on any upstream blocking issues

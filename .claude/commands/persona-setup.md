# /persona-setup

Initializes a persona session for the ldc-demo project. Run this at the start of every persona session to verify your environment, review your scope, and either pick up an open issue or create a new feature worktree.

## Step 1 — Identify your persona

Ask the user which persona they are invoking, or infer it from the current working directory:

| Current directory ends with... | Persona |
|---|---|
| `losant-device-demo` (main clone) | merge-manager or triage |
| `losant-device-demo-worktrees/security` | security |
| `losant-device-demo-worktrees/gitops-manager` | gitops-manager |
| `losant-device-demo-worktrees/docs` | docs |
| `losant-device-demo-worktrees/qa` | qa |
| `losant-device-demo-worktrees/product-designer` | product-designer |
| `losant-device-demo-worktrees/development` | developer or test-engineer |

If the directory is `development`, the active persona is either developer or test-engineer — ask the user which one if not obvious from context.

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

## Step 5 — Create the feature branch (developer and test-engineer only)

**For all other personas:** worktrees already exist and have a fixed branch — skip to Step 6.

**For developer and test-engineer:** you are already in the correct worktree (`development/`). After picking an issue, create a feature branch from `origin/main` within this same session:

```bash
git fetch origin
git checkout -b feature/developer/<short-descriptor> origin/main
# e.g. git checkout -b feature/developer/state-registry origin/main

# Confirm branch
git branch --show-current
```

The test-engineer starts their own separate Claude session from this same `development/` directory, then checks out the same branch:

```bash
# The test-engineer runs in their own terminal:
cd ~/projects/losant-device-demo-worktrees/development
# Then inside Claude: git checkout feature/developer/<short-descriptor>
```

**Do not create a new worktree directory per feature.** The `development/` worktree is permanent — switch branches within it for each new feature.

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

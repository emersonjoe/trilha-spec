# Issue tracker: GitHub + `.trilha/`

GitHub Issues is the source of demand, scope, priority and public discussion. Use the `gh` CLI for issue-tracker operations.

Implementation specifications live in `.trilha/specs/` (written with `trilha-spec spec new`) and the task breakdown and its state in `.trilha/tasks/`; `docs/adr/003-fonte-da-verdade.md` says why that is the only source of truth and what `.specify/` and the frozen `specs/NNN/` folders still are. Do not duplicate the complete implementation plan in the GitHub issue.

Link the two surfaces explicitly: the spec's `issue:` field references its originating issue, and the issue or pull request references the relevant spec path and `TASK-NNN` identifiers.

## Conventions

- Create issues with `gh issue create`.
- Read issues and comments with `gh issue view <number> --comments`.
- List and filter issues with `gh issue list --json number,title,body,labels,comments`.
- Comment with `gh issue comment`, update labels with `gh issue edit`, and close with `gh issue close`.
- Infer the repository from `git remote -v`; commands run inside this clone.

## Pull requests as a triage surface

**PRs as a request surface: no.**

## Skill operations

- When a skill says to publish to the issue tracker, create a GitHub issue containing the problem, expected outcome and acceptance boundary.
- When a skill says to fetch a ticket, use `gh issue view <number> --comments`, then follow its `.trilha/specs/` reference for implementation details.
- Use the spec in `.trilha/specs/` for the technical specification and plan; use the tasks in `.trilha/tasks/` for executable task state, dependencies and evidence (adr/003).
- Wayfinder maps use `wayfinder:map`; child tickets use `wayfinder:research`, `wayfinder:prototype`, `wayfinder:grilling`, or `wayfinder:task`.
- Use native GitHub sub-issues and dependencies when available; otherwise record explicit issue links in the body.

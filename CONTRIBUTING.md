# Contributing Guide

Everything in this repository (code, docs, commits, PRs, changelog) is written in **English**.

## Types

Every commit and every PR must be classified under **one** of the types below. In commit messages and PR titles
the type is written in **lowercase** (e.g. `feat`). Elsewhere in the docs it may appear in uppercase for emphasis.

| Type          | When to use                                                  |
|---------------|--------------------------------------------------------------|
| `feat`        | New feature                                                  |
| `fix`         | Bug fix                                                      |
| `refactor`    | Code change that does not alter behavior                     |
| `performance` | Performance improvement                                      |
| `style`       | Formatting, whitespace, lint (no logic change)               |
| `docs`        | Documentation only                                           |
| `test`        | Adding or adjusting tests                                    |
| `build`       | Build system, dependencies, packaging                        |
| `ci`          | Pipelines and continuous integration configuration           |
| `chore`       | Maintenance tasks that fit none of the above                 |
| `revert`      | Reverts a previous commit                                    |

## Commits

We follow [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) with a lowercase type:

```
type(scope): description

optional body: what and why

optional footer: BREAKING CHANGE, Closes #123
```

Rules:

- Description in the imperative mood, lowercase, no trailing period, up to 72 characters.
- The scope is optional and names the affected area (e.g. `agent`, `simulator`, `world`).
- The body explains **why** the change was made, not just what.
- Breaking changes: add `!` after the type/scope (`feat!:`) and/or a `BREAKING CHANGE:` footer.
- One commit = one logical change.

Examples:

```
feat(agent): add resource search behavior
fix(simulator): fix infinite loop when removing an agent
refactor(world): extract collision calculation into its own module
performance(engine): use spatial hash for neighbor detection
feat(api)!: rename `id` field to `agentId`
```

## Pull Requests

- **Title**: same format as commits, `type(scope): description`.
- **One type per PR**. If a change spans more than one type, split it into separate PRs.
- **A descriptive body is required**, following the template in
  [`.github/PULL_REQUEST_TEMPLATE.md`](.github/PULL_REQUEST_TEMPLATE.md):
  context, what changed, how to test, and impact.
- Keep PRs small and focused; link the related issue.
- Prefer squash merging, using the PR title as the commit message.

## Changelog

The [`CHANGELOG.md`](CHANGELOG.md) follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
and [Semantic Versioning](https://semver.org/).

- Every PR that changes user-visible behavior must add an entry under `[Unreleased]`.
- Write for people who use the project, not for people reading the code: describe the effect, not the implementation.
- Internal changes with no visible effect (`style`, `test`, `ci`, `chore`) usually do not belong here.
- On release, `[Unreleased]` becomes `[X.Y.Z] - YYYY-MM-DD` and a new `[Unreleased]` section is created.

Mapping between commit/PR type, changelog section, and version bump:

| Type            | Changelog section  | Version bump          |
|-----------------|--------------------|-----------------------|
| `feat`          | Added              | minor                 |
| `fix`           | Fixed              | patch                 |
| `refactor`      | Changed            | patch (if visible)    |
| `performance`   | Changed            | patch                 |
| `docs`          | Changed            | only if relevant      |
| `revert`        | Removed / Changed  | patch                 |
| Breaking (`!`)  | Changed / Removed  | major                 |

Allowed sections: `Added`, `Changed`, `Deprecated`, `Removed`, `Fixed`, `Security`.

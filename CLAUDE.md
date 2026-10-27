# Agent Instructions

## Language

Everything in this repository must be written in **English**: code, comments, identifiers, documentation,
commit messages, PR titles and descriptions, and changelog entries.

## Commits and Pull Requests

Always follow [CONTRIBUTING.md](CONTRIBUTING.md):

- Commits use Conventional Commits with a lowercase type: `type(scope): description`
  (e.g. `feat(agent): add resource search behavior`).
- Allowed types: feat, fix, refactor, performance, style, docs, test, build, ci, chore, revert.
- PRs use the same format for the title, a single type, and a complete description based on
  `.github/PULL_REQUEST_TEMPLATE.md` (context, what changed, how to test, impact).

## Changelog

For user-visible changes, add an entry under `[Unreleased]` in `CHANGELOG.md`, in the section matching the
type (see the mapping in `CONTRIBUTING.md`).

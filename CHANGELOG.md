# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to
[Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- Commit, pull request, and changelog standards (`CONTRIBUTING.md`, PR template).
- Simulation engine that runs models defined by a `*.schema.json`, an optional `*.context.json`,
  `*.agent.json` files (single agents or generated populations) and Lua scripts.
- Decisions with eligibility, propensity (as a native expression or a Lua function), precondition,
  probabilistic outcome and success/failure consequences.
- Atomic effects with constraints (`min`, `max`, enums), conservation-preserving transfers and
  configurable merge policies for shared context fields.
- Agent memory, private/public attributes and hidden context fields, relations, groups and 2D space.
- Reproducible runs: the same seed gives the same result regardless of the number of workers.
- `agentsim validate`, `agentsim run` (multiple seeds in parallel, aggregated statistics) and
  `agentsim jsonschema` (editor autocompletion for agent and context files).
- Outputs: JSONL event log with levels and sampling, per-tick metrics CSV, JSON snapshots.
- Example models: prisoner's dilemma, entrepreneurship with a central bank, a bread island with supply
  and demand for perishable food and scarce land (posted prices and an order book, hunger, ageing,
  inheritance, and random droughts, golden ages and epidemics), and a one-million-agent benchmark.
- Modeling guide and architecture documentation.
- `scenario-builder`, a Claude Code skill that turns a real-life situation into a model, with agent
  behavior grounded in how real actors behave, a library of behavioral formulas and a review checklist.
- A `README.md` for every example model, describing its agents, context, decisions, rules and metrics;
  `agentsim validate` warns when a model has no `README.md`.

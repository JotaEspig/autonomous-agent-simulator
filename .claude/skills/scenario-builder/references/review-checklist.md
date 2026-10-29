# Model review checklist

Go through this before reporting results of a new or changed scenario.

## Behavior

- [ ] Every agent type has a written behavior specification (state, perception, goals, actions,
      decision rules, constraints).
- [ ] Each decision rule matches how the real actor behaves, and the README says which real
      mechanism it encodes.
- [ ] No rule encodes an aggregate outcome (equilibrium, target population, expected trend).
- [ ] Agents only use information they could have: own state and memory, public context,
      observable neighbors.
- [ ] Needs come before discretionary choices where that is how people behave.
- [ ] Nothing a real agent would obviously do is impossible (e.g. eating one's own produce).
- [ ] Heterogeneity comes from distributions, not from identical agents.

## Mathematics

- [ ] Every formula has units, bounds and a direction of effect that matches reality.
- [ ] No quantity can grow without bound when it has a real limit.
- [ ] Rates are converted to the tick length.
- [ ] No self-referential loop (an agent's price anchored on prices it set) unless that is the
      phenomenon being studied.
- [ ] Every parameter has a source or a stated assumption. None was chosen to make the output
      look good.

## Mechanics

- [ ] The order of rules and decisions within a tick follows the real sequence of events.
- [ ] Markets price trades the way the real mechanism does.
- [ ] Money and goods are conserved, or created and destroyed only where reality does it
      (production, consumption, spending outside the model).
- [ ] Prices and offers can respond when nobody buys or sells.
- [ ] Edge cases are handled: empty markets, death, bankruptcy, the last unit of a resource.

## Verification

- [ ] `agentsim validate` passes; warnings are understood.
- [ ] Micro checks in a scratch copy confirm each behavior's direction.
- [ ] Results come from many seeds; the report shows the spread, not one run.
- [ ] Surprising results were traced back to agent behavior before being reported.
- [ ] The README describes behavior, results, realism and limitations.

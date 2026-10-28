# Prisoner's dilemma

An iterated prisoner's dilemma between players with different personalities. The default
setup pits a **trickster** (low honesty, high greed) against an **honest** player (high
honesty, but resentful when betrayed). The question: how do personality and memory shape
cooperation over repeated rounds?

```sh
agentsim run models/prisoners-dilemma --seeds 20
```

## How a round works

Each tick is one round, and both moves are simultaneous:

1. **Matchmaking** (pre rule): players are paired at random through the `opponent` relation.
2. **Decision**: each player chooses `cooperate` or `defect` based on its personality and its
   memory of the opponent's past moves, then writes its move to `ctx.moves.<own id>`.
3. **Settle** (post rule): reads both moves, pays each player from the payoff matrix, stores the
   opponent's move in each player's memory, and clears the moves.

Players never act on each other directly. Their moves go through the context, and the
`settle` rule applies the consequences.

## Context

| Field             | Type                 | Default | Meaning                                           |
|-------------------|----------------------|---------|---------------------------------------------------|
| `temptation`      | float                | 5       | T: defecting against a cooperator                 |
| `reward`          | float                | 3       | R: both cooperate                                 |
| `punishment`      | float                | 1       | P: both defect                                    |
| `sucker`          | float                | 0       | S: cooperating against a defector                 |
| `moves`           | map of `"C"`/`"D"`   | empty   | Moves of the current round, keyed by agent id      |
| `cooperationRate` | float                | 0       | Share of cooperative moves in the last round      |

The payoffs keep the classic order T > R > P > S, so defecting is always individually better in
a single round, while mutual cooperation beats mutual defection. Change them in
`pd.context.json`.

## Agents

One type, `player`:

| Attribute  | Visibility | Meaning                                                        |
|------------|------------|----------------------------------------------------------------|
| `honesty`  | private    | Baseline inclination to cooperate (0 to 1)                     |
| `greed`    | private    | Temptation to exploit an opponent who has been trusting        |
| `grudge`   | private    | How strongly betrayals reduce cooperation                      |
| `score`    | public     | Accumulated payoff                                             |
| `betrayed` | private    | Rounds in which it cooperated and the opponent defected        |

Memory: `opponentMoves`, the last 10 moves of its opponents.

| Agent file                    | honesty | greed | grudge |
|-------------------------------|---------|-------|--------|
| `agents/trickster.agent.json` | 0.2     | 0.9   | 0.2    |
| `agents/honest.agent.json`    | 0.9     | 0.05  | 0.6    |

## Decisions

Both decisions are eligible only when the player has an opponent this round. A player always
makes one of the two moves; there is no idle option.

- **`cooperate`**: propensity is a native expression,
  `honesty × (1 − grudge × share of "D" in memory)`, so betrayals erode cooperation in
  proportion to the player's grudge.
- **`defect`**: propensity is computed in Lua as
  `(1 − honesty) + greed × trust + grudge × (1 − trust)`, where `trust` is the share of "C" in
  memory (0.5 with no history). Greedy players exploit trusting opponents, and resentful
  players retaliate.

The move is chosen by roulette in proportion to the two propensities.

## Rules

| Rule          | Phase | Script                    | What it does                                        |
|---------------|-------|---------------------------|-----------------------------------------------------|
| `matchmaking` | pre   | `scripts/matchmaking.lua` | Re-pairs players at random; with an odd count, one sits out |
| `settle`      | post  | `scripts/settle.lua`      | Pays the payoffs, updates memories and `betrayed`, sets `cooperationRate`, clears `moves` |

## Metrics

`trickster.score`, `honest.score`, `score.mean`, `cooperationRate`, plus the automatic
population and decision counters (`decision.player.cooperate.chosen`, ...).

## Results

With the default setup, over 20 seeds and 100 rounds, the trickster ends with about 250 points
and the honest player with about 105. The honest player's memory ends up full of "D": it learns
to retaliate, but too late to catch up.

## Things to try

- Add more players with a population file (`"count": 100` and distributions for the
  personality attributes). Matchmaking already handles any number.
- Raise `sucker` or lower `temptation` and watch cooperation rise.
- Give the honest player a higher `grudge` (a tit-for-tat-like strategy).

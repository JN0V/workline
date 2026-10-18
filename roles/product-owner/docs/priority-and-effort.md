# Priority and effort, proposed

Part of [the product owner](../README.md). Proposed by
[ADR-0040](../../../docs/adr/0040-priority-and-effort-are-the-forges-the-product-owner-estimates-them.md),
built a step at a time ([status](status.md#missing-in-the-order-to-build-it)):

- **Built, off by default**: the names below read; where each value is
  kept found when the settings ask
  ([`priority`, `effort`](settings.md#priority-and-effort)); the effort
  estimated, each value set once; the grid's order, Next's cells, the
  Urgents said apart, estimates stuck on one value said.
- **Not built yet**: GitHub's fields, GitLab's weight, workline's labels
  renamed. Unset, with `order` on, the role sets its own labels
  `workline:priority/1` to `/4`.

Below, what the proposal does on real cases.

- **Two values per issue**: a priority (Urgent, High, Medium, Low) and an
  effort, how hard it is to build (Low, Medium, High).
- **Kept where the forge keeps them**: a GitHub organisation's fields,
  GitLab's labels and weight, or the project's own labels.
- **Estimated by the role**, only where empty, Urgent included; a person
  checks and corrects with the forge's own tools.
- **Off by default**: on for workline first, then for every project at
  `normal` once the measures pass there and on a test repository.

## Other scales, read by name

The role reads a project's values by their names, never by where an
option sits in a list.

| A project has | Read as |
|---|---|
| GitHub's Priority field: Urgent, High, Medium, Low | the same |
| GitHub's Effort field, its options listed High, Medium, Low | by name: High sorts last, though listed first |
| `P0`, `P1`, `P2`, `P3` | Urgent, High, Medium, Low |
| `P1` to `P4`, with no `P0` | Urgent, High, Medium, Low |
| `P0` to `P5` | Urgent, High, Medium; `P3` to `P5` Low |
| `priority::1` to `::4`, GitLab's own convention | Urgent, High, Medium, Low |
| `priority/critical-urgent`, `important-soon`, `important-longterm`, `backlog` | Urgent, High, Medium, Low |
| `P-critical`, `P-high`, `P-medium`, `P-low` | Urgent, High, Medium, Low |
| `effort1: hours`, `effort2: days`, `effort3: weeks` | effort Low, Medium, High |
| `effort/small`, `medium`, `large`, `x-large` | effort Low, Medium, High, High |
| GitLab's weight: 0 or 1; 2 or 3; 4 or more | effort Low; Medium; High |
| `priority/awaiting-more-evidence`, or any name no table knows | no value, said once in the summary |
| `size/XS` to `size/XXL` | never read: on Kubernetes they count a pull request's lines |

- **A name no table knows** is mapped in the project's settings.
- **GitLab's weight**, when the role writes it: 1, 3 or 5, values found
  on both common point scales (1, 2, 3, 5, 8, and 1 to 5).

## Where the values were found

Said once in the night's summary, for example, the first two built:

```markdown
Priority: labels priority/*, created. Effort: labels effort/*, created when first set.
Priority: labels P0 to P3, the project's. Effort: read nowhere, its setting unset.
Priority: GitHub's field Priority. Effort: GitHub's field Effort.
Priority: labels priority::*, created. Effort: GitLab's weight.
```

- **Created labels** on GitHub take the fields' option names, so a
  repository moved to an organisation maps them one to one.
- **On GitLab Free**, the `priority::1` to `::4` labels the role creates
  are ranked: GitLab's own "Label priority" sort then lists Urgent first.

## Next

A night's Next on a project whose values are labels: five ready issues
in the backlog's order, each with its cell of the grid.

```markdown
Next to build
1. #202 Export a month — High · Effort Low → quick win
2. #188 Retry a failed upload — High · Effort Medium
3. #175 Rewrite the importer — High · Effort High → big bet
4. #210 A typo in the help — Low · Effort Low → fill-in
5. #96 Dark mode — no priority, effort not estimated

3 of 9 ready issues have no priority: they follow their number.
```

- **The cells named**: High with Low effort, quick win; High with High,
  big bet; Low with Low, fill-in; Low with High, time sink. The others
  show their two values only.
- **An issue that waits on another** still comes after it, whatever its
  cell.

Each Urgent the role set that night is listed apart, ready or not, for a
person to see at once; too many Urgents also trip the flag on estimates
stuck on one value:

```markdown
Set to Urgent tonight, to check: #214 Exports drop the last row of a file
```

On a public repository whose fields are visible to organization members
only, Next keeps the order and hides the values:

```markdown
Next to build
1. #202 Export a month
2. #188 Retry a failed upload
Values hidden: the fields are visible to organization members only.
```

## A person's correction

| A person | The role, the next night |
|---|---|
| set a value the role did not set | keeps it |
| changed the role's effort from Medium to High | keeps High; never writes that issue's effort again; counts it as corrected |
| cleared the role's priority | leaves it empty; never writes that issue's priority again |
| corrected the role's effort on 3 issues (`undone-max`) | proposes effort on every issue instead of setting it |
| added `workline:accepted` to a proposal, at `cautious` | sets the values proposed |

- **One axis at a time**: a corrected effort says nothing of the
  priority.
- **Estimates stuck on one value** are said in the summary: "Effort:
  Medium on 8 of the 11 issues the role estimated."

## workline's own labels

| Today | Renamed to |
|---|---|
| `workline:priority/1` (never created) | `priority/urgent`, created when first needed |
| `workline:priority/2` | `priority/high` |
| `workline:priority/3` | `priority/medium` |
| `workline:priority/4` | `priority/low` |

- **A rename** keeps each issue's label, notifies nobody and changes no
  state: the role still knows each value as its own.

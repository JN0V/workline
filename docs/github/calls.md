<!-- workline
sources: [internal/forge/retry.go, internal/forge/cli.go, internal/forge/budget.go, internal/line/budget.go]
checked: 76e290d9
-->
# A GitHub call that fails for a moment

From [workline on GitHub Actions](../ci-github.md#what-the-jobs-may-do).

GitHub can answer a call with a 502, refuse it for its rate, or not
answer at all, for a moment. workline calls GitHub through `gh`, which
has no option to try a call again, nor a timeout
([#491](https://github.com/JN0V/workline/issues/491)). workline tries
such a call again before it calls the forge unreachable, by the same
rule as [on GitLab](../gitlab/calls.md):

- **A read** (a GET, a GraphQL query) is tried again when it gets no
  answer within 60 s, its connection drops or is refused, or GitHub
  answers a 5xx (but 501), a 429, or a 403 for its rate limit.
- **A write** (a POST, PATCH, PUT, DELETE, a GraphQL mutation) is tried
  again only when GitHub certainly did not take it: no connection could
  be made, or GitHub refused it for its rate.
- **A write answered late, or with a 5xx, is not sent again**: GitHub
  may have taken it, and sending it twice could post a comment twice.
  The failure says so: "a write: not sent again, GitHub may have taken
  it". Run it again later
  ([troubleshooting](../troubleshooting.md#token-caps-and-the-agent)).
- **3 tries in all**, 2 s then 4 s apart. The wait GitHub asks is waited
  for instead, up to 60 s; a longer one is not, said:
  - its `Retry-After`;
  - for a rate limit spent, its `X-RateLimit-Reset`;
  - when it asks nothing, a minute, as
    [GitHub's docs](https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api)
    ask.
- **A run of the line** (`workline route issue`, `route reconcile`)
  waits longer for a rate limit spent, up to an hour, its roles' calls
  too: its CI job can wait, where a push cannot
  ([the API budget](#the-api-budget)).
- **Never tried again**: a certificate refused, any other 4xx.
- **After the last try**, the failure says so: "(tried 3 times)".

How workline reads what `gh` gives:

- **The status and the wait asked**: `gh` is asked for each answer's
  headers (`--include`), taken out of what is read.
- **No answer**: `gh` prints each answer as it comes, a page's too; one
  that prints nothing for 60 s is stopped.
- **A host not reached**: `gh` does not tell a host that does not exist
  from a name not resolved for a moment. Nothing was sent: tried again.
- **Many pages**: a read that fails on one is read again from the first.

## The API budget

A CI job's own token (`GITHUB_TOKEN`) has
[1,000 REST calls an hour](https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api)
for the whole repository; a busy hour of the line of states can spend
them ([#804](https://github.com/JN0V/workline/issues/804)).

- **Read first, at no cost**: a run of the line asks GitHub's
  `rate_limit`, which counts against no limit, before it reads an issue.
- **Under a tenth left**, of the REST calls or the GraphQL points: the
  run waits for the reset, up to an hour, then goes on. Its summary and
  a warning (`api-budget-low`) say how much was left and how long it
  waited.
- **Spent half-way**: a call refused for the rate waits for the reset
  as well, said in the job's log.
- **GitLab** says its budget only in each answer's headers: a run there
  reads nothing first.

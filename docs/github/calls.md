<!-- workline
sources: [internal/forge/retry.go, internal/forge/cli.go, internal/forge/budget.go, internal/line/budget.go]
checked: 4eaa098f
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
  waits longer for a rate limit spent, up to 5 minutes, its roles' calls
  too ([the API budget](#the-api-budget)).
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
- **Under a tenth left**, of the REST calls or the GraphQL points, the
  reset within 5 minutes: the run waits for it, then goes on. Its
  summary and a warning (`api-budget-low`) say how much was left and
  how long it waited.
- **The reset further off: deferred.** The run reads and runs nothing,
  and ends with no failure. Its summary and a warning
  (`api-budget-deferred`) give the reset's time. The issue's step stays
  due: the hourly reconciler runs it after the reset.
- **Why 5 minutes**: a job that waits holds a runner. On a busy hour,
  twenty runs waiting would hold every runner a free plan gives, and
  the pull requests' tests would queue behind them.
- **Spent half-way**: a call refused for the rate waits for a reset
  within 5 minutes too, said in the job's log; a later one fails the
  step as the forge out of reach, and the reconciler runs it again.
- **The line's alone**: the project's tests, checks and agents run
  without this wait, so a test of the project never reads it
  ([#940](https://github.com/JN0V/workline/issues/940)), nor the line's
  other variables ([the list](../environment.md#the-lines-own)).
- **GitLab** says its budget only in each answer's headers: a run there
  reads nothing first.

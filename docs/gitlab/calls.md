<!-- workline
sources: [internal/forge/retry.go, internal/forge/gitlab.go]
checked: 2d24380f
-->
# A GitLab call that fails for a moment

From [workline on GitLab CI](../ci-gitlab.md#what-the-template-does).

gitlab.com can answer late, or not at all, for a moment: a product
owner's run of about 80 calls lost its plan to one notes call
([#489](https://github.com/JN0V/workline/issues/489)). workline tries such
a call again before it calls the forge unreachable:

- **A read** is tried again when it gets no answer within 60 s, its
  connection drops or is refused, or GitLab answers a 5xx (but 501) or
  429 (too many requests).
- **A write** is tried again only when GitLab certainly did not take it:
  no connection could be made, or GitLab answered 429.
- **A write answered late, or with a 5xx, is not sent again**: GitLab may
  have taken it, and sending it twice could post a note twice. The run
  ends `blocked-external`, said: "a write: not sent again, GitLab may have
  taken it". Run it again later
  ([troubleshooting](../troubleshooting.md#token-caps-and-the-agent)).
- **3 tries in all**, 2 s then 4 s apart. A `Retry-After` GitLab sends is
  waited for instead, up to 60 s; a longer one is not, said.
- **Never tried again**: a host that does not exist, a certificate
  refused, any other 4xx.
- **After the last try**, the failure says so: "(tried 3 times)".

The rules are those of GitLab's own Go client, which tries again a 429,
a 5xx and a call that could not be sent, with two changes: a read that
timed out is tried again (the case seen), and a write answered with a 5xx
is not. GitHub's calls, through `gh`, go by the same rule
([on GitHub](../github/calls.md)).

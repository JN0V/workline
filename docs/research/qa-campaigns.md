# QA campaigns on a running build — focused research (2026-11-04)

**Question.** A release candidate is cut; who checks that the application
behaves right for a real user, on concrete cases, with evidence? How do
others plan a campaign, play it in a simulated environment, hand the rest
to a person on real hardware, and turn failures into bugs?

In the ecosystem's words: *test plan*, *test run*, *test execution*,
*test campaign*, *release validation*, *test matrix*, *sign-off*,
*blocker bug*, *exploratory testing*, *charter*, *session sheet*,
*hardware-in-the-loop* (HIL), *device farm*, *virtual scene*, *recording
and playback*, *screenshot testing*, *golden image*, *needle*. Read for
[ADR-0062](../adr/0062-the-qa-role-plays-a-campaign-on-a-running-build.md).

## Conclusions first

- **Borrow the shape everyone shares**: a plan of cases, one *run* per
  build and environment, one result per case (pass, fail, blocked, not
  run), a failed case linked to a bug. Kiwi TCMS, Fedora's matrices and
  Ubuntu's tracker all hold it.
- **Borrow the split automated / manual on one matrix**: Fedora's openQA
  fills what it can, people fill the rest of the same page; a release
  ships when the matrix holds no blocker.
- **Borrow evidence by default**: openQA keeps a screenshot a step,
  Wokwi CI a screenshot and the serial log, Roborazzi a compare image.
  A result without its evidence is a claim.
- **Borrow the simulated input**: the Android emulator's virtual scene
  and webcam feed, ARCore's recorded sessions, Wokwi's simulated ESP32.
  Each is a recipe the project owns; none is guessed.
- **Borrow the AI explorer's discipline**: Playwright's planner writes a
  Markdown plan before any test; Firebase's App Testing agent asserts
  each step on the screen, caches what passed, and says it may act
  differently each run.
- **What workline must add**: the campaign derived from the milestone's
  named scenarios (no tool reads them); the results kept on the forge
  (a check run and an issue, no test management server); a person's
  part as a checklist with a timeout; the judgement "does the observed
  behaviour meet the Need", which no replay tool gives.

## Test plans, runs and results

| Where | Plan | Run | A result | A failure |
|---|---|---|---|---|
| [Kiwi TCMS](https://kiwitcms.readthedocs.io/en/latest/guide/testrun.html) | a Test Plan, "a high-level container" of cases | a Test Run per build and environment | a Test Execution: status, comments, bugs, one per case | a bug linked to the execution |
| [Fedora release validation](https://fedoraproject.org/wiki/Test_Results:Fedora_33_RC_1.1_Desktop) | test cases by milestone and environment, on a wiki matrix | a page per compose (RC, nightly) | pass, warn, fail, inprogress; each tagged with who (a person, `openqa`) | a Bugzilla bug, proposed as a release *blocker* |
| [Ubuntu ISO QA tracker](https://iso.qa.ubuntu.com/) | test cases a product, products a milestone | the milestone (an event) | passed or failed, with a comment | a failed result must link at least one Launchpad bug |
| [GitLab quality management](https://gitlab.com/groups/gitlab-org/-/epics/617) | test cases (Markdown), in suites | a test session copies suites to run | a session fails if one case fails | proposed; sessions from pipelines asked in [#8766](https://gitlab.com/gitlab-org/gitlab/-/issues/8766) |

- **A run is per build**: Fedora's rule, a test "must have been run
  against the release candidate build" for the milestone to be approved.
- **A finished run is frozen**: Kiwi TCMS's status goes Running to
  Finished and cannot change after.
- **The forges have no test run**: GitLab's test cases are a partial
  start; GitHub has none. Both have milestones, releases, issues with
  task lists, and check runs with a summary and annotations.

## Exploratory testing in sessions

[Session-based test management](https://www.satisfice.com/download/session-based-test-management)
(Jonathan and James Bach, 2000):

- **A charter** per session: "Explore *area* with *tools* to discover
  *information*".
- **Time-boxed**, about 60 to 120 minutes.
- **A session sheet**: charter, areas, data files, test notes, bugs,
  issues (problems with the testing itself), then a debrief.
- What to take: an explorer's run is bounded and leaves a written
  sheet; "issues" (the harness could not reach a screen) are kept apart
  from bugs.

## Android: emulators, camera input, screenshots

| Tool | What it gives |
|---|---|
| [Emulator camera](https://developer.android.com/studio/run/emulator-use-camera) | a *virtual scene* back camera (`hw.camera.back = virtualscene`) with PNG or JPEG images placed in it; a host webcam as the camera, which on Linux can be a `v4l2loopback` device fed by ffmpeg from a video file |
| [ARCore recording and playback](https://developers.google.com/ar/develop/recording-and-playback) | a session recorded to MP4 with its sensor data, played back in place of the camera; Google warns trackables and poses "may differ" in playback |
| [android-emulator-runner](https://github.com/ReactiveCircus/android-emulator-runner) | an emulator in GitHub Actions; [KVM on 2-vCPU Linux runners](https://github.blog/changelog/2024-04-02-github-actions-hardware-accelerated-android-virtualization-now-available/) since 2024-04; without KVM it falls back to `-accel off`, slowly and quietly |
| [Roborazzi](https://github.com/takahirom/roborazzi) | screenshots on the JVM with Robolectric; record, verify, compare tasks; a `_compare.png` and an HTML report |
| [Firebase App Testing agent](https://firebase.google.com/docs/app-distribution/android/app-testing-agent) | natural-language goals in YAML, each step with a final-screen assertion; caches the actions of a passing run, falls back to AI when a replay fails; cannot see a `FLAG_SECURE` screen |

- **The camera is the hard part**: the emulator feeds images or a
  stream; what the app's detector makes of them is only seen in the
  app's own logs, overlays and metrics. A debug build reading frames
  from a file is the project's seam, not a tool's.
- **Playback is not the world**: ARCore says so; a campaign keeps a
  real-device row for what playback cannot prove.

## Embedded: simulation and hardware-in-the-loop

| Tool | What it gives |
|---|---|
| [Wokwi CI](https://docs.espressif.com/projects/esp-idf/en/stable/esp32/third-party-tools/wokwi.html) | an ESP32 simulated in CI: `wokwi-cli` with a timeout and an expected serial line, a screenshot kept as an artifact; `pytest-embedded-wokwi` for suites; a token as a CI secret |
| [labgrid](https://github.com/labgrid-project/labgrid) | boards driven from pytest (`--lg-env`): power, serial, flashing, boards on other machines through a coordinator; a library, not a framework |
| [openQA](https://open.qa/docs/) | whole systems driven in VMs; *needles* (a screenshot and the regions that must match) decide each step; a screenshot kept per step |

- **Simulation covers logic, not the board**: radio, power, sensors and
  timing stay a person's, or a lab's, as Fedora keeps its hardware rows
  for people.
- **A lab is the project's**: labgrid needs boards on a machine; a
  project without one gives a person a checklist.

## AI agents exploring an application

| Where | How | Measured |
|---|---|---|
| [Playwright test agents](https://playwright.dev/docs/test-agents) (1.56+) | *planner* explores the running app, writes a Markdown plan under `specs/`; *generator* writes tests from it; *healer* repairs failing ones | — |
| [DroidAgent](https://arxiv.org/abs/2311.08649) (ICST 2024) | a planner sets tasks, an actor plays them as GUI events | 61% activity coverage against 51%; 317 of 547 tasks it made judged realistic |
| [GPTDroid](https://arxiv.org/abs/2310.15780) | page information to the model, actions back, feedback in a loop | 32% more activity coverage, 48 new bugs, 25 confirmed |
| [LLM-Explorer](https://arxiv.org/abs/2505.10593) | the model only where judgement pays | 9 to 148 times cheaper than the two above |

- **Plan first, in text**: Playwright's planner writes the plan before
  any action; a person reads it.
- **An agent's run varies**: Firebase says so; caching what passed
  makes the next run a replay.
- **Cost is the limit**: LLM-Explorer's gain is keeping the model out
  of routine steps.

## What workline takes

- **The campaign is a test run on the forge**: one per release
  candidate; its cases the milestone's named scenarios; its results a
  check run on the candidate's tag and a comment on the candidate's
  issue; a failure, a bug in the milestone.
- **Automated and manual rows on one campaign**, as Fedora's matrix: the
  simulated rows played by the role, the hardware rows a checklist for a
  person.
- **Evidence a row**: screenshot, log excerpt, metric, as openQA and
  Wokwi keep them; no evidence, no pass.
- **The harness is the project's recipe**: emulator and frame source,
  Wokwi, a lab; the role asks for one when missing, never builds it
  silently.
- **The agent plans and judges, the recipe plays**: a plan in Markdown
  first, as Playwright's planner; replay what passed, as Firebase.
- **Not taken**: a test management server (Kiwi TCMS, TestRail): the
  forge's issues and check runs hold the same states.

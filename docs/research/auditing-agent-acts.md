# Auditing a sample of agent acts — focused research (2026-10-14)

**Question.** How do others re-check a sample of decisions an automated
system took alone: how many to draw, who reads them again, how a
disagreement is settled, and how an error rate is reported? For the
auditor widened to every role's acts
([#205](https://github.com/JN0V/workline/issues/205),
[#276](https://github.com/JN0V/workline/issues/276)).

**Verdict.**

- **Draw by stratum, the same draw on a rerun.** Auditors sample each
  population (here: each role and kind of act) apart, so a rare kind is
  seen; tracing tools draw by a hash of the item's id, so every reader
  draws the same.
- **A second reader who does not see the first answer, then a third
  on disagreement.** Clinical trials read images so; corpus annotation
  calls it double annotation and adjudication. The adjudicator picks one of the
  two readings.
- **A panel of different models beats one large judge** and is cheaper;
  a model favours its own outputs.
- **An error rate is reported with its uncertainty**, per kind, and its
  sample size says how far it can be trusted: about 30 items with no
  error bound a rate under 10%, at 95%.
- **What workline adds**: the proof, a quote the engine finds again at the
  act's commit; nobody else checks the second reader's evidence
  mechanically.

## Drawing the sample

| Practice | What it does | To borrow |
|---|---|---|
| Audit sampling, tests of controls ([a public audit guide's table](https://hudoig.gov/sites/default/files/2024-02/appendix.pdf)) | Attribute sampling: the size follows the confidence wanted, the tolerable deviation rate and the rate expected. At 95% and none expected: 65 items for a 5% tolerable rate, 35 for 10% | A sample size that says which rate it can bound; few items bound little |
| Stratified sampling ([Wei, Zufall, Jia, AIES 2024](https://arxiv.org/abs/2305.09601)) | Recall of automated moderation estimated far cheaper by sampling per stratum | One draw per role and kind, rounded up |
| Deterministic sampling ([OpenTelemetry's `TraceIdRatioBased`](https://opentelemetry.io/docs/specs/otel/trace/sdk/)) | "The sampling algorithm MUST be deterministic": a hash of the trace's id decides, so every participant draws the same | Already done: a digest of the week and the act ([ADR-0015](../adr/0015-the-weekly-sample-is-written-on-the-forge.md)) |

- **The rule of three**: no error in *n* items bounds the rate under about
  3/*n* at 95%. One draw a week of a role's rare kind needs months to say
  anything; counts, not shares, until then
  ([ADR-0019](../adr/0019-the-line-evaluates-itself.md)).

## Reading it again, independently

| Practice | What it does | To borrow |
|---|---|---|
| The IIA's Three Lines Model ([Mayer Brown on the 2020 update](https://www.mayerbrown.com/en/insights/publications/2020/07/the-blurred-lines-of-organizational-risk-management)) | The first line acts; the second oversees; the third, internal audit, "performs independent and objective assessments", apart from management | The auditor reports; the process engineer proposes ([roles panorama](roles-panorama.md)) |
| Blinded independent central review ([Applied Clinical Trials](https://www.appliedclinicaltrialsonline.com/view/discordance-between-bicr-readers)) | Two independent readers; a third, the adjudicator, selects "the most correct assessment". Over 40 oncology trials, the two disagreed on 31% of progression dates | The second reading blind to the first; disputes settled by choosing a reading |
| Double annotation ([Artstein and Poesio, 2008](https://www.cl.cam.ac.uk/teaching/1516/R216/Artstein.pdf)) | Agreement corrected for chance (Cohen's kappa, Krippendorff's alpha); disagreements adjudicated, or the item dropped; low agreement blames the guidelines first | Agreement per kind of act: a low one points at the question asked |
| Panel of LLM evaluators ([Verga et al., 2024](https://arxiv.org/abs/2404.18796)) | Several smaller models of disjoint families outperform one large judge, with less intra-model bias, over seven times cheaper | Disputes to three models, not one |
| Self-preference ([Panickssery et al., NeurIPS 2024](https://arxiv.org/abs/2404.13076)) | A model recognises and favours its own outputs | Never the model that acted ([ADR-0005](../adr/0005-independence-takes-the-best-level-available.md)) |

## Reporting an error rate

- **The EU's Digital Services Act**, Art. 15(1)(e), asks platforms for
  "indicators of the accuracy and the possible rate of error" of their
  automated moderation; most reports give a bare number
  ([DSA Observatory](https://dsa-observatory.eu/?p=2400)). A rate without
  its sample and its method is read as noise.
- **Precision and recall, not accuracy**, when errors are rare: the
  population is lopsided (the judge's 173 refusals to 56 shows,
  [the judge, measured](judge-measured.md)).
- **An interval for a small sample**: Wilson's, as the judge's measure
  gave (8 of 20: a fifth to three fifths).

## Already measured on workline

[The judge, measured](judge-measured.md) ran that design by hand on
2026-10-14: four raters on a third model, blind to the judge's verdict;
9 disagreements of 40 to a panel of three agents, each given both
readings as "a" and "b"; unanimous on truth for all 9. The auditor
widened is that method, run every week by the engine.

## What workline takes

| Borrowed | Where in [ADR-0046](../adr/0046-the-auditor-re-checks-every-roles-acts.md) |
|---|---|
| Stratified draw, one in ten per role and kind, by a hash | which acts are drawn |
| A blind second reading, on another model | how one act is re-checked |
| Adjudication by choosing a reading, by a panel of models | kept and counted |
| Counts per kind with their interval; a sample size that bounds a rate | how the role leaves beta |

Not borrowed: a dashboard or a server for the verdicts. The forge's
issue, its comments and reactions hold them, as the weekly sample already
does ([ADR-0015](../adr/0015-the-weekly-sample-is-written-on-the-forge.md)).

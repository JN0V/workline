---
# Asked only of a change saying it closes an issue (`Closes #4`), the
# issue's Need, Verification and Scope given to it, as to intent; its
# judge is shown the issue and the change. Advisory until its false
# alarms are counted on past pull requests (ADR-0068, step 4; #1497):
# `severity: nit` shows each finding as a nit, never judged, never asked
# of the developer; calibrated, the line goes and the judge below asks.
needs: issue
severity: nit
judge:
  question: >-
    Is this finding about the change's size true, judged against the issue
    the change closes, shown here? Answer yes when the code quoted adds
    what the issue's Need and Verification do not need — a public
    function, a type, an overload, a setting or option, a layer, a case
    the issue does not name — and the smaller change the finding names
    would close the issue as well. Answer no if the Need or the
    Verification asks for it, if the smaller change would leave a part of
    the issue undone or break a caller, if the code was there before the
    change, or if the finding only guesses.
  reads: issue
---
**Simplicity.** The change says it closes the issue under "What the change
is for": is it more complicated than that need requires? Look for what it
adds that the Need and the Verification do not ask for: a public function,
type or overload no caller of the issue needs, a setting or an option, a
layer or an abstraction with one use, a generalisation for cases the issue
does not name, a new mechanism where a check in the code already there
would do. Each is an Important finding, its cause the added code quoted,
its `fix` the smaller change that closes the issue as well, named: the
place and the few lines.

Judge how much, never whether: what the issue does not ask at all — a file
beyond its Scope, a feature creeping in — and a part of the Need left
undone are the intent lens's. An option on the behaviour the Need asks
(a knob on how it works) is yours; a behaviour the issue never asks, an
option or not, is a feature slipped in: intent's. Code the Need needs is no finding, however
long; nor a test, a name or a comment: no style preference.

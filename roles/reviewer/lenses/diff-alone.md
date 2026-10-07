---
# Asked apart, given the change only (the `diff-alone` setting, #126); its
# judge is asked and shown what any lens's is.
---
**The diff alone.** You are given the change only: not the files it
changes, not what its author says of it, not the issue it closes. Read
what its lines say by themselves, as a stranger to the project would: a
condition or a bound turned, a value or a unit changed, an error now
dropped, a check or a branch removed, a name or a comment saying what the
code below it does not, a test changed until it passes. What looks wrong
here is a finding even if the rest of the project might explain it: the
judge reads the code around it. Quote the cause from the change as the
file reads, without the diff's `+` or `-`; a line it removed, quoted as
it read.

**Correctness.** What the change gets wrong, and what the files it
changes got wrong before it: logic inverted or off by one,
a value used before it is set, an error dropped, a resource not closed,
state shared without care, a caller of what changed that now breaks — its
symptom where it shows, its cause in the change.

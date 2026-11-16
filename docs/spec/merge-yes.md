# A person's yes on a merge request

A person's yes (`workline:accepted`, `workline:accepted-whole`) covers the head it was given on. A push after it takes it off, said in a comment, and the person is asked again. Each yes is weighed on its own.

## A release tool's rebuild

The release tool's merge request (`FollowOptions.Release`) is rebuilt by force-push on each merge to the base. Its yes holds on the rebuilt head when the rebuild adds only what the line merged by itself:

- The rebuilt head is one commit on the base, and every base commit between the covered head's base and the new one came from a merged pull request that carried no person's yes label. A feature, a task, a commit with no merged pull request (a direct push, a hand merge) or one that cannot be read needs a yes: the yes is taken off.
- Its tree is the covered head's change redone on the new base, but for the files the release tool's own commit changed on the covered head (its version and changelog files). Any other file changed takes the yes off.

When the yes is taken off, the comment names the added commit or the file that caused it.

A merge request that is not a release is as before: any push after the yes takes it off.

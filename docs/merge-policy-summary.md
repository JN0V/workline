---
sources:
  - internal/engine/merge.go#mergePolicy
  - internal/engine/merge.go#heldByConversations
  - internal/engine/merge.go#personsCommentOpen
checked: 3f432ab14
verified: agent:claude-code
---
# What the merge summary says

Part of [the merge policy](merge-policy.md): the words `workline follow`
writes in its job's summary, per pull request, all said by `merge.go`.
At `normal`, more words:
[what the summary says there](merge-policy-normal.md#what-the-summary-says).

## Each word

| Said | When |
|---|---|
| `auto-merge-on`, `merged-at-once` | accepted, its head's verdict passed; at `normal`, "with no person's yes", its kind; "merged now", an auto-merge the forge left |
| `merge-refused` | the forge refused to merge now one whose auto-merge it left: auto-merge left on |
| `persons-comment-open`, `comments-unread` | a person's comment no role answered, nor their own later yes: nothing merges; or its comments could not be read |
| `held-by-conversations`, `conversations-unread` | auto-merge on, checks green, not merged: the open conversations that hold it, or why they could not be read |
| `verdict-kept`, `verdict-missing`, `review-to-come` | the verdict kept; one not judged yet; marked ready, the draft's verdict still on its head |
| `verdict-red`, `review-short`, `checks-red` | nothing turned on, why; a person's yes does not lift a red verdict |
| `review-read-whole` | a review not whole, taken by the reading of who put `workline:accepted-whole` |
| `accepted-reset` | a push after the yes: the yes and auto-merge off |
| `vetoed`, `draft`, `auto-merge-off` | the forge's veto, or the label taken off |
| `label-ignored` | put by someone who may not push nor merge, the label named |
| `merge-unguarded`, `merge-token`, `merge-unsupported` | a base, a token or a forge it cannot act with |
| `checks-unread` | its head's checks could not be read: only the line's verdict counts |
| `merge-state-unread`, `verdict-unread` | its state or its head's verdict could not be read: left as it is |
| `auto-merge-refused`, `label-not-made` | the forge refused to turn auto-merge on, or to make the label |
| `refusal-unsaid` | a refused merge, said on the request and its issue (a conflict's files named), could not be written there |

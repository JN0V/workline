<!-- workline
sources: [ci/gitlab/workline.gitlab-ci.yml, internal/engine/follow.go, internal/engine/settle.go, Dockerfile]
checked: fbf84ebb
-->
# Signed commits on GitLab

From [workline on GitLab CI](../ci-gitlab.md#a-project-that-refuses-unsigned-commits).

The push rule *Reject unsigned commits* refuses every unsigned commit
pushed, the bot's too: a doc fix, a merge request rebuilt by `follow`,
one brought up to date by the [merge policy](../merge-policy.md)
([#549](https://github.com/JN0V/workline/issues/549)). Give the bot an
SSH key: the template signs its commits with it.

## Set it up

1. **A key for the bot**, made on your machine, with no passphrase: a job
   cannot type one.

   ```sh
   ssh-keygen -t ed25519 -N '' -C workline -f workline-signing
   ```

2. **The private key as a File variable**, `WORKLINE_SIGNING_KEY`: not
   protected, as the token (a merge request's pipeline needs it). GitLab
   masks no value of several lines: only who may change a pipeline can
   read it. With glab, logged in as a Maintainer:

   ```sh
   glab variable set WORKLINE_SIGNING_KEY --type file --raw --repo <group/project> < workline-signing
   ```

   Or Settings → CI/CD → Variables → Add variable, type **File**.
3. **Delete `workline-signing`** from your machine once stored.

## What the template does

- **Each job that pushes** (`workline:apply`, `workline:merge`,
  `workline:follow`, gardening's, the release docs', the sample's) sets
  git to sign with the key: `gpg.format ssh`, `user.signingkey`,
  `commit.gpgsign true`.
- **The key file is fixed first**: readable by its owner alone (GitLab
  writes it readable by all, which `ssh-keygen` refuses), and ended with
  a newline, without which `ssh-keygen` cannot read it.
- **Every commit it pushes is then signed**: the roles' commits, and the
  commits `follow` and the merge policy write, as git is set. workline
  never picks a key of its own.
- **A key that cannot sign** fails the job, said, nothing pushed. A key
  given as a plain variable fails it too, the key never printed.
- **No variable**: nothing changes.
- **The image** holds `ssh-keygen` (`openssh-client`) from v0.38.0.

## Shown as verified, too

The push rule takes any signature. GitLab shows a commit **Verified**
only when its signing key is the bot's and the commit carries the bot's
own address:

- register the public key on the bot, with its own token (no admin right
  needed): `POST /user/keys`, `usage_type=signing`;
- the template writes `workline@noreply.<host>`, not the bot's address:
  its commits show as signed by another user.

## A GPG or X.509 key

- The image has no GnuPG: GitLab's rule takes an SSH signature as it takes
  a GPG one.
- With an image of your own holding `gnupg`, import the key in the jobs
  and set git (`gpg.format`, `user.signingkey`, `commit.gpgsign`): workline
  signs as git is set.

## Without the template

A [trigger tool's job](../gitlab-trigger.md) or your own: set git the same
way before `workline apply` or `workline follow`.

Tried with a local remote that refuses unsigned commits; not tried on a
GitLab yet.

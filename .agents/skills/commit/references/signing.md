# Sign-off and Cryptographic Signing

This project requires two separate things on every commit, per
`.github/PULL_REQUEST_TEMPLATE.md`:

1. **DCO sign-off** (`Signed-off-by:`) — the committer's attestation, under
   the [Developer Certificate of Origin v1.1](https://developercertificate.org/),
   that they wrote the code or otherwise have the right to submit it under
   the project's Apache-2.0 licence. Added with `-s`/`--signoff`.
2. **Cryptographic signature** — proves the commit really came from the
   signer. Not mandated by the PR template, but this machine has it enabled
   globally, so every commit gets one automatically.

```sh
git commit -s -m "..."
# -s adds Signed-off-by; the cryptographic signature is applied
# automatically because commit.gpgsign=true is set globally.
```

## Checking the local signing setup

```sh
git config --get commit.gpgsign   # true → signing happens automatically
git config --get gpg.format       # "ssh" on this machine
git config --get user.signingkey  # the SSH public key used to sign
```

If `commit.gpgsign` is not `true` in a given environment, add `-S`
explicitly:

```sh
git commit -s -S -m "..."
```

## GPG setup (alternative to SSH signing)

```sh
git config --global gpg.format openpgp
git config --global user.signingkey <GPG key ID>
git config --global commit.gpgsign true
```

## SSH setup (used on this machine)

```sh
git config --global gpg.format ssh
git config --global user.signingkey /path/to/key.pub
git config --global gpg.ssh.allowedsignersfile ~/.ssh/allowed_signers
git config --global commit.gpgsign true
```

## Verifying a commit

```sh
git log --show-signature
git verify-commit <sha>
```

<!--
  Pull Request Template
  Thanks for contributing! Fill in each section below.
  Delete any section that isn't relevant, but keep the checklist complete.
-->

## Summary

<!-- One sentence: what does this PR do? -->

## Why

<!-- Why is this change needed? Link any related issue(s). -->

Closes #<!-- issue number, or "N/A" -->

## What changed

<!-- Brief description of the approach and key changes. -->

-

## How to validate

```sh
make provider
make test
make lint
```

<!-- Add any additional manual steps here. -->

## Impact

- [ ] No breaking changes
- [ ] Breaking change — describe below:

---

## Checklist

### Code quality

- [ ] `make test` passes locally
- [ ] `make lint` passes locally
- [ ] New behaviour is covered by tests

### Documentation

- [ ] Public provider changes are reflected in the schema
- [ ] README updated if public API changed
- [ ] If this PR was created or filled by an AI agent, indicate it here and confirm a human reviewed the changes.

### Commits

- [ ] Commits follow [Conventional Commits](https://www.conventionalcommits.org/) with a required scope
      (`provider` | `sdk` | `examples` | `tests` | `docs` | `ci` | `deps` | `tooling`)
- [ ] Each commit is focused and self-contained

### Legal

- [ ] **DCO** — every commit includes a `Signed-off-by` trailer
      (`git commit --signoff`, or `git commit -s`).
      By signing off you certify that you wrote the code and have the right
      to submit it under the project's [Apache-2.0 licence](../blob/main/LICENSE),
      per the [Developer Certificate of Origin v1.1](https://developercertificate.org/).
- [ ] **CLA** — if you are contributing on behalf of an employer or under a
      different copyright, ensure your organisation has agreed to the project's
      CLA (or open a discussion with the maintainer first).

# Commit Body Guide

## Good body — explains WHY

```text
The Random resource's Create returned before the provider confirmed
the value was persisted, so a crash between the API call and the
state write left orphaned resources with no Pulumi record.

Waiting for the confirmation response was chosen over a background
reconciliation loop because it keeps Create synchronous and matches
how every other resource in this provider already behaves.
```

## Bad body — describes the diff, not the reason

```text
Added a wait after the API call in random.go. Updated the Create
function signature. Added a test for the new behaviour.
```

## Full commit examples

### Feature with scope and body

```text
feat(provider): Add RandomComponent for grouped random resources

Users creating several related random values (e.g. a username and
a password) had to declare each Random resource separately and wire
outputs by hand. A component resource groups them under one logical
unit, matching the pattern used by other multi-resource providers.

Assisted-by: claude-code:claude-sonnet-5
Signed-off-by: Alexandre NICOLAIE <xunleii@users.noreply.github.com>
```

### Fix, short body

```text
fix(sdk): Correct nullable handling in Config.cs

Config.Get() returned an empty string instead of null when a key was
unset, which made downstream null-coalescing checks in generated SDKs
silently mask misconfiguration instead of failing fast.

Assisted-by: claude-code:claude-sonnet-5
Signed-off-by: Alexandre NICOLAIE <xunleii@users.noreply.github.com>
```

### Trivial change, no body needed

```text
chore(deps): Bump github.com/pulumi/pulumi/sdk/v3 to v3.150.0
```

## References to issues or PRs

Write them as plain text rather than a bare `#N`, since the header parser
reserves `#` for its own trailing-reference capture and a stray `#` mid-body
can be misread by tooling that also runs conventional-changelog parsers:

| Instead of   | Write                  |
| ------------ | ------------------------ |
| `fixes #42`  | `fixes issue 42`         |
| `see #17`    | `see pull request 17`    |

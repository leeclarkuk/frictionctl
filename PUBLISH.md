# Publish the Real Platform Proof

This branch is a drop, not a frictionctl change. Do not merge it to main.
Do not apply the patch. Fetch this bundle so HEAD stays 3522e48.

Expected commit: `3522e485f4b35c8fdc9d5d53e25aabe4858c0701`

From a clone of `leeclarkuk/platform-engineering-reference` that can push:

```bash
git clone --depth 1 --branch cursor/real-platform-proof-bundle-0a0e \
  https://github.com/leeclarkuk/frictionctl.git /tmp/rpp-bundle

git fetch /tmp/rpp-bundle/real-platform-proof.bundle HEAD:cursor/real-platform-proof-0a0e
git checkout cursor/real-platform-proof-0a0e
test "$(git rev-parse HEAD)" = 3522e485f4b35c8fdc9d5d53e25aabe4858c0701
git push -u origin cursor/real-platform-proof-0a0e
```

Open a PR to `main`. CI must pin `frictionctl@v0.1.0` and the Real Platform
Proof job must stay green. If it does, merge. Do not cut a frictionctl
v0.2.0 release.

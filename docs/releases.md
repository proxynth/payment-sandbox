# Releases

Payment Sandbox uses semantic versioning for published application releases.
Release tags must use the `vMAJOR.MINOR.PATCH` form, for example `v0.1.0`.

## Create a release

Start from an up-to-date `main` checkout and make sure the working tree is
clean. Review the changes since the previous release, then create and push an
annotated tag:

```bash
git switch main
git pull --ff-only origin main
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0
```

Pushing the tag starts the `Release` GitHub Actions workflow. The workflow
validates the tagged commit, then GoReleaser publishes a GitHub Release with:

- macOS, Linux and Windows archives for `amd64` and `arm64`;
- SHA-256 checksums;
- generated release notes based on the commits since the previous tag.

Normal pushes and pull requests do not publish releases.

## Retry a failed release

A failed release must be retried with the same tag. Open the failed run in
GitHub Actions and use **Re-run failed jobs** (or **Re-run all jobs** after
fixing an infrastructure issue). Do not create a second tag for the same
version and do not move the existing tag to another commit.

The release configuration keeps existing release notes and replaces an asset
when a retry encounters an already-uploaded artifact. This makes a retry
idempotent for the original tagged commit while preserving a single release
for the version.

If the failure is caused by the source itself, create a corrective commit on
`main` and publish the next patch version instead of reusing the tag.

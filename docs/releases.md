# Releases

Payment Sandbox uses semantic versioning for published application releases.
Release tags use the `vMAJOR.MINOR.PATCH` form, for example `v1.1.0`.

## Repository setup

The release workflow expects a repository secret named
`RELEASE_PLEASE_TOKEN`. Configure it with a GitHub token that can create and
update pull requests, write release contents and create release tags. Using a
dedicated token allows the generated release pull request and its CI checks to
trigger normally; events created with the default `GITHUB_TOKEN` do not start
new workflow runs.

Keep the token scoped to this repository and rotate it according to the
repository's security policy. The workflow continues to use the least
privilege `GITHUB_TOKEN` for GoReleaser publication after the release has been
created.

## Create a release

Release creation is automated from merges to `main`:

1. Release Please creates or updates a release pull request when releasable
   Conventional Commits are present.
2. Review and merge that release pull request when the version and generated
   notes are correct.
3. The workflow creates the version tag automatically, without creating a
   GitHub release through Release Please.
4. The workflow validates the tagged commit and GoReleaser creates the
   immutable GitHub release and uploads the application artifacts in the same
   run. A manually pushed version tag can also trigger the publication path.

The current release baseline is `1.1.0`, as configured in
`.release-please-manifest.json`. Subsequent versions are derived from
Conventional Commit types:

| Commit | Version impact |
| --- | --- |
| `fix:` | Patch |
| `feat:` | Minor |
| `feat!:` or a `BREAKING CHANGE` footer | Major |

The generated GitHub release contains:

- macOS, Linux and Windows archives for `amd64` and `arm64`;
- SHA-256 checksums;
- generated release notes based on the commits since the previous release.

Normal feature pushes and pull requests do not publish releases. Do not create
or push version tags manually as part of the normal workflow.

## Retry a failed release

A failed validation should first be retried from its existing GitHub Actions
run using **Re-run failed jobs** or **Re-run all jobs** after fixing an
infrastructure issue. Keep the same release pull request, tag and commit.

The workflow creates tags only for merged Release Please release PRs. GoReleaser
is the only component that creates the GitHub release. This avoids attempting
to update an immutable release after Release Please has created it. If
GoReleaser has already created a release for a tag, do not rerun publication
against that tag; publish a new corrective version instead.

If the release pull request contains an incorrect version or changelog, fix
the underlying Conventional Commit history or configuration, then let Release
Please update the existing release pull request. Do not create a second tag
for the same version or move an existing tag to another commit.

The release configuration keeps existing release notes and replaces an asset
when a retry encounters an already-uploaded artifact. This only applies before
the GitHub release becomes immutable. If the failure is caused by the source
itself after a release has been published, create a corrective commit on
`main` and publish the next patch version instead of reusing the tag.

The release workflow can also be started manually with **Run workflow**. This
is a recovery or diagnostic mechanism; it does not replace merging the
generated release pull request.

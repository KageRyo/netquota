# Contributing to NetQuota

Thanks for helping improve NetQuota. It is an open-source project, and small, reviewable changes are preferred.

## Required workflow: GitHub Flow

Every contribution—including code, tests, documentation, and CI changes—must follow [GitHub Flow](https://docs.github.com/en/get-started/using-github/github-flow). Contributors must work on a short-lived branch and open a pull request; do not commit or push directly to `main`.

1. **Create a branch.** Start from the latest `main` and use a short, descriptive branch name:

   ```sh
   git switch main
   git pull --ff-only origin main
   git switch -c feat/upload-quota
   ```

   Good names describe the change, such as `feat/upload-quota`, `fix/state-persistence`, or `docs/readme-markdown`. If you contribute from a fork, push the branch to your fork and open the pull request against `KageRyo/netquota:main`.

2. **Make focused changes.** Keep each branch limited to one coherent change. Keep accounting behavior in the core packages rather than calculating usage in the tray layer. Add or update focused unit tests for behavior changes, especially date, counter, quota, and persistence edge cases.

3. **Run the required checks locally.** Before opening a pull request, run:

   ```sh
   gofmt -w .
   gofmt -l .
   go vet ./...
   go test ./...
   go build ./cmd/netquota
   ```

   `gofmt -l .` must produce no output. Do not include generated binaries, user data, or machine-specific settings in the change.

4. **Commit and push the branch.** Use [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) and keep commits logically isolated:

   ```text
   feat(quota): add an upload limit
   fix(storage): preserve alert marks across restart
   test(usage): cover counter rollover
   docs: clarify local counter limitations
   ```

   Push the branch to the remote:

   ```sh
   git push --set-upstream origin feat/upload-quota
   ```

5. **Open a pull request.** Target `main`, complete the pull request template, describe what changed and why, link related issues, and report the checks you ran. The CI checks must pass before requesting merge.

6. **Address review comments on the same branch.** Continue committing and pushing updates to the pull request until the requested changes are resolved. Keep the pull request focused and up to date with `main`.

7. **Merge and clean up.** A maintainer merges the pull request after review and passing checks. After it is merged, delete the branch. Do not bypass the pull request process by pushing to `main`.

## Scope and compatibility

NetQuota supports normal-user operation on Windows and Linux desktop environments. Avoid adding a server, account system, packet capture requirement, or cloud dependency without first discussing the scope in an issue. New platform-specific code should use build constraints and keep the shared core portable.

## Reporting problems

Use the issue templates for reproducible bugs and focused feature proposals. Do not attach personal network identifiers, configuration files, or state files unless they have been sanitized.

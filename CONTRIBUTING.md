# Contributing to NetQuota

Thanks for helping improve NetQuota. It is an open-source project, and small, reviewable changes are preferred.

## Before opening a change

1. Read the README and design notes.
2. Keep accounting behavior in the core packages; do not make the tray layer calculate usage.
3. Add or update unit tests for behavior changes, especially date, counter, quota, and persistence edge cases.
4. Run the local checks:

   ~~~sh
   gofmt -w .
   go vet ./...
   go test ./...
   go build ./cmd/netquota
   ~~~

5. Keep user data, generated binaries, and machine-specific settings out of commits.

## Commit messages

Use Conventional Commits, for example:

~~~text
feat(quota): add an upload limit
fix(storage): preserve alert marks across restart
test(usage): cover counter rollover
docs: clarify local counter limitations
~~~

See https://www.conventionalcommits.org/en/v1.0.0/ for the commit format. Keep the subject short and describe the user-visible or repository-level change.

## Scope and compatibility

NetQuota supports normal-user operation on Windows and Linux desktop environments. Avoid adding a server, account system, packet capture requirement, or cloud dependency without first discussing the scope in an issue. New platform-specific code should use build constraints and keep the shared core portable.

## Reporting problems

Use the issue templates for reproducible bugs and focused feature proposals. Do not attach personal network identifiers, configuration files, or state files unless they have been sanitized.


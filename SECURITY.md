# Security policy

## Supported versions

Entpager is pre-v1. Only the latest tagged release receives security fixes.
Users should upgrade to the newest release before reporting a problem that may
already be resolved.

The module requires Go 1.24, matching the minimum required by the latest stable
Ent release. Applications should run on a currently supported Go release with
the latest security patch. Standard library vulnerabilities are corrected by
upgrading the Go toolchain rather than by changing this dependency-free module.

| Version | Supported |
| --- | --- |
| Latest tagged release | Yes |
| Older releases | No |

## Report a vulnerability

Do not disclose suspected vulnerabilities in a public issue, discussion, pull
request, or social-media post.

Use
[GitHub private vulnerability reporting](https://github.com/hcarriz/entpager/security/advisories/new)
to send the report privately. Include, when available:

- the affected version and Go version;
- a minimal reproduction or proof of concept;
- the expected and observed behavior;
- the security impact and realistic attack scenario; and
- any suggested mitigation or workaround.

Reports will be acknowledged and investigated as soon as practical. The
maintainer will coordinate validation, remediation, release timing, and public
disclosure with the reporter. Please allow a reasonable remediation period
before publishing details.

## Scope

Relevant reports include uncontrolled resource consumption, unsafe limit or
offset bypasses, integer overflow, panics reachable through untrusted pagination
input, and vulnerabilities in repository automation or release artifacts.

Entpager limits work within a single pagination operation. Applications remain
responsible for authentication, authorization, rate limiting, request and
database timeouts, query complexity, deterministic ordering, and deployment
resource limits.

## Maintainer requirements

Security-sensitive changes require regression tests and fuzzing. Before a
release, maintainers should run the full test suite, race detector, and pinned
analysis tools:

```sh
go vet ./...
go test -race ./...
go tool -modfile=tools/go.mod staticcheck ./...
go tool -modfile=tools/go.mod govulncheck ./...
```

The required CI workflow runs static analysis, a vulnerability scan, and a short
fuzz pass. The scheduled Security workflow repeats vulnerability analysis and
CodeQL scanning. Before a release or after a security-sensitive change,
manually dispatch the Maintenance workflow for its ten-minute fuzz pass and
repeat vulnerability scan. Preserve any minimized failure as a regression seed
or corpus entry after reviewing it for sensitive data.

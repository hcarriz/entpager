# Security policy

## Supported versions

Entpager is pre-v1. Only the latest tagged release receives security fixes.
Users should upgrade to the newest release before reporting a problem that may
already be resolved.

The module remains source-compatible with Go 1.21, but applications should run
on a currently supported Go release with the latest security patch. Standard
library vulnerabilities are corrected by upgrading the Go toolchain rather than
by changing this dependency-free module.

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

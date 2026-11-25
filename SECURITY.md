# Security Policy

## Supported versions

| Version | Supported |
|---|---|
| `v0.1.0` / latest `main` | ✅ |
| Earlier semver releases (`v0.x.y`) | ✅ for critical fixes |
| Older commits / forks | ❌ |

MayaMCP is a template and library; fixes land on `main` first and are then
backported to `ngodingmcp`. Please upgrade to the latest commit or tag before
reporting an issue.

## Reporting a vulnerability

Please **do not** open a public issue for a security problem. Use GitHub's
private vulnerability reporting:

<https://github.com/neflalabs/mayamcp/security/advisories/new>

Include:

- the affected version/commit and module path;
- a minimal reproduction (tool call, arguments, and transport);
- the impact you observed (for example, a secret appearing in an activity log);
- whether the issue is in `api/`, `internal/mcpserver/`, or `cmd/mayamcp/`.

You should receive an acknowledgement within a few days. We will coordinate a
fix and a release before public disclosure.

## Scope notes

- The core deliberately has **no auth/TLS**. Deployments must provide it at
  the application or reverse-proxy layer. Reports about that design choice are
  not vulnerabilities; reports about accidental exposure caused by the core
  are.
- The activity log is designed never to contain commands, file contents,
  tokens, or secrets. A regression there is treated as high severity.
- Tool contract changes (`tools/list`, golden file, budget) are reviewed via
  the same process as security fixes when they affect privacy guarantees.

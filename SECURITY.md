# Security Policy

## Reporting a vulnerability

Please **do not** open a public GitHub issue for security problems.

Report privately through GitHub: **Security → Report a vulnerability**
(private vulnerability reporting), or email the maintainer listed on the
repository owner's profile.

Include:

- affected package and version or commit (e.g. `modules/auth @ v0.3.0`);
- steps to reproduce or a proof of concept;
- impact as you understand it.

We aim to acknowledge reports within 3 working days and to ship a fix for
confirmed high-severity issues within 14 days. Please give us that time
before disclosing publicly.

## Supported versions

Only the latest release on `main` receives security fixes.

## Secrets

If you find a credential in this repository, report it the same way. Test
fixtures intentionally use obviously fake values such as
`sk_test_not-a-real-key`.

Internal handling rules: [docs/standards/SECURITY_STANDARD.md](docs/standards/SECURITY_STANDARD.md).

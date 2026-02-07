# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |

## Reporting a Vulnerability

If you discover a security vulnerability in CompactMapper, please report it responsibly.

**Do NOT open a public GitHub issue for security vulnerabilities.**

Instead, please use [GitHub's private vulnerability reporting](https://github.com/mmitucha/compactmapper/security/advisories/new) to submit a report.

You can expect:
- Acknowledgment within 48 hours
- An assessment and response plan within 7 days
- A fix or mitigation within 30 days for confirmed vulnerabilities

## Scope

CompactMapper processes CSV files from Intelligent Compaction equipment and converts them to LAS point cloud format. Security considerations include:

- **Dependencies**: Third-party Go module vulnerabilities

## Security Measures

- Automated dependency updates via Dependabot
- CodeQL static analysis on every push and PR
- Dependency review on pull requests
- `govulncheck` in CI pipeline

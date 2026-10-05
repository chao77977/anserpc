# Security Policy

## Supported Versions

anserpc is pre-1.0. Security fixes are applied to the latest released minor
version and to `main`.

| Version | Supported |
| --- | --- |
| latest `0.x` | ✅ |
| older        | ❌ |

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, report them privately using GitHub's
[private vulnerability reporting](https://docs.github.com/en/code-security/security-advisories/guidance-on-reporting-and-writing-information-about-vulnerabilities/privately-reporting-a-security-vulnerability)
on this repository (Security tab → "Report a vulnerability"), or contact the
maintainer directly.

Please include:

- A description of the vulnerability and its impact
- Steps to reproduce (a minimal proof of concept if possible)
- Affected version(s) or commit

## What to Expect

- Acknowledgement of your report as soon as practical.
- An assessment of the issue and a plan for a fix.
- Coordinated disclosure: we will agree on a timeline before any public
  disclosure, and credit you in the release notes if you wish.

## Dependency Advisories

CI runs `govulncheck`. Note that it may report advisories in the Go standard
library tied to the toolchain version used by the CI runner; these are tracked
separately from vulnerabilities in anserpc's own code or its direct
dependencies.

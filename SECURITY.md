# Security Policy

Private-to-Public Release Gate is designed to reduce two specific risks: sensitive private context entering a generated public tree, and an unreviewed difference between that generated tree and its reviewed distribution. It is one control in a publication process, not a general-purpose data-loss-prevention system or a substitute for human review.

## Supported versions

| Version | Security support |
|---|---|
| Latest release | Supported |
| `main` | Supported for evaluation; may contain unreleased changes |
| Older releases | Upgrade to the latest release before reporting behavior already changed there |

Security fixes are released from the protected `main` branch. Published binaries include SHA-256 checksums on the corresponding GitHub release.

## Reporting a vulnerability

Use this repository's **Report a vulnerability** option under the GitHub Security tab. Do not open a public issue for a suspected vulnerability, bypass, private-data exposure, or credential leak.

Include only what is necessary to reproduce the problem:

- affected release, tag, or commit;
- operating system and architecture;
- the relevant policy shape, using synthetic values;
- expected and observed behavior;
- minimal reproduction steps;
- the security impact and plausible attack path.

Do **not** submit a real token, credential, private policy, personal path, internal hostname, private email address, customer identifier, or proprietary source tree. Replace sensitive material with a synthetic value that triggers the same behavior. If a real credential may have been exposed, rotate or revoke it through the issuing provider before investigating the scanner behavior.

Reports will be acknowledged and triaged on a best-effort basis. Confirmed issues will be scoped, fixed on a private security branch when appropriate, validated with a regression test, and disclosed through a release or GitHub security advisory. No response-time SLA is offered.

## Security model

The gate builds a candidate distribution in a temporary directory, applies explicitly allowlisted overlay files, scans generated regular-file paths and contents, and compares the result with a reviewed distribution checkout.

The intended sequence is:

1. exclude private or control-plane paths;
2. reject overlay files that are not explicitly allowlisted;
3. generate the candidate public tree outside both checkouts;
4. scan generated paths and regular-file contents for configured privacy indicators and supported token patterns;
5. compare content, entry type, executable-bit state, and symlink targets with the reviewed distribution;
6. return a nonzero exit status until privacy and drift findings are resolved.

Privacy findings report the affected relative path and rule name only. The matched private value is deliberately suppressed.

## Trust boundaries and assumptions

The source checkout, distribution checkout, policy file, local operating system, and user account running the gate are trusted inputs. The tool is intended for a maintainer reviewing repositories they control.

The gate:

- reads regular-file contents from the source and distribution trees;
- creates and removes a temporary generated tree using the host operating system;
- reproduces symlinks as symlinks and compares their targets;
- does not execute repository files, hooks, build scripts, or overlay content;
- does not modify the source or distribution checkout;
- does not contact a remote service.

Do not run the tool with elevated privileges or against an untrusted repository. It is not a sandbox, malware scanner, or safe parser for hostile filesystem trees.

## Enforced controls

The current implementation enforces:

- safe relative policy paths for exclusions and overlay entries;
- an explicit allowlist for every overlay file;
- failure when an allowlisted overlay file is missing;
- exclusion of `.git` metadata from generation and comparison;
- non-allowlisted email detection in paths and regular-file contents;
- non-allowlisted macOS `/Users/<name>` path detection;
- local Mac hostname detection;
- case-insensitive configured forbidden-term detection;
- selected GitHub and OpenAI-style token-pattern detection;
- path, content, entry-type, executable-bit, and symlink-target drift detection;
- deterministic, path-sorted findings;
- exit code `1` for findings and exit code `2` for operational or configuration errors.

## Known limitations

Treat a passing result as evidence that the configured checks passed—not as proof that a tree contains no sensitive information.

- Pattern matching is intentionally bounded and cannot recognize every credential format, identity, hostname, proprietary term, encoded value, archive, or binary secret.
- Secret-like token scanning currently covers selected patterns, not every provider.
- Forbidden terms are only as complete as the private policy supplied by the operator.
- Symlink targets are compared for drift but are not privacy-scanned, and symlink destinations are not traversed.
- Regular files are read as bytes and converted for pattern matching; encrypted, compressed, encoded, or otherwise transformed sensitive data may not be detected.
- The gate does not verify that either checkout is clean, at an expected commit, on a particular branch, or synchronized with its remote.
- The gate does not inspect Git history. Sensitive data removed from the working tree may still exist in earlier commits.
- The gate does not publish, approve, merge, or attest a release. Those remain separate reviewed actions.
- A malicious actor who can change the policy, source, overlay, distribution, executable, or CI configuration may be able to weaken the control.

Use Git secret scanning and push protection, protected branches, required checks, review of policy changes, and provider-side credential rotation alongside this tool.

## Operator responsibilities

Keep the real publication policy in the private canonical repository. Do not commit real private terms, account identifiers, host inventories, credentials, or internal paths to this public repository or its fixtures.

Before publishing:

1. verify both checkouts are clean and pinned to the intended commits;
2. review every new exclusion and overlay allowlist entry as a publication decision;
3. run the gate against the exact candidate trees;
4. investigate operational errors separately from expected drift findings;
5. manually inspect the generated change and public diff;
6. run repository-native tests and code scanning;
7. publish only after the gate returns zero privacy and zero drift findings.

Do not solve a privacy finding by broadly allowlisting the detected identity or term unless that value is intentionally public and the decision has been reviewed.

## Supply-chain and release integrity

GitHub Actions dependencies are pinned to full commit SHAs and updated through Dependabot. CI runs with read-only repository contents permission. The protected `main` branch requires the repository validation check and blocks force-pushes and deletion.

Release consumers should download artifacts from this repository's GitHub Releases page and verify them against the published `SHA256SUMS` file. Checksums detect accidental or malicious artifact changes after publication; they are not a cryptographic identity signature.

## Scope examples

Security-relevant reports include:

- bypassing an overlay allowlist;
- escaping the source, generated, or distribution root through policy paths;
- exposing matched private values in output or errors;
- failing open after a privacy or drift finding;
- traversing a symlink unexpectedly;
- missing a documented supported token or privacy pattern;
- release artifacts that do not correspond to the documented source revision.

General feature requests, documentation corrections without security impact, and requests to add a new detection category may be filed as public issues if they contain no sensitive information.

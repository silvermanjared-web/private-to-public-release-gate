# Private-to-Public Release Gate

![Private-to-Public Release Gate](assets/private-to-public-release-gate.png)

A small Go reference implementation for publishing a reviewed distribution from a private canonical repository without treating “the trees match” as proof that the content is safe to publish.

## The problem

Private repositories often contain the strongest operating code and the most sensitive context. A public mirror introduces two different failure classes:

1. **Publication drift:** the generated artifact and reviewed public repository diverge.
2. **Privacy drift:** the generated artifact faithfully carries a private path, email, hostname, organization term, or credential into public history.

A hash or tree comparison catches the first and completely misses the second. This gate checks both before release.

## Release contract

```mermaid
flowchart LR
  A["Private canonical source"] --> B["Explicit exclusions"]
  B --> C["Allowlisted public overlay"]
  C --> D["Generated tree"]
  D --> E["Strict privacy scan"]
  E --> F["Git-representable drift comparison"]
  F --> G["Reviewed public distribution"]
```

The pipeline fails closed when:

- a source path has no explicit export decision;
- the overlay contains an unreviewed file;
- generated paths or contents contain non-allowlisted emails or macOS home users;
- generated content contains local hostnames, configured private terms, or token-like material;
- content, entry type, executable bit, or symlink target differs from the reviewed distribution.

Privacy findings expose only the path and rule. Matched private values are suppressed.

## Field validation

This pattern was first exercised against a real private canonical governance repository and its reviewed sanitized distribution. The initial detector run reported **56 path-level findings**: **25 content differences**, **7 distribution-only entries**, and **24 canonical-only exclusions**. Each path received an explicit direction decision: preserve the reviewed distribution version through the allowlisted overlay, retain a distribution-only public surface, or exclude a canonical-only private control-plane entry. After those decisions were encoded, the same detector reached **zero drift** and enforcement was re-enabled.

That original distribution has since been archived after the reusable publication pattern was extracted into this repository. The archived consumer is no longer an active product, but the reconciliation result remains the operational evidence behind this reference implementation: the gate was used to turn a noisy first measurement into a reviewed, enforceable zero-drift contract against a real private system.

## Quick start

Copy `publication-policy.example.json` into the private canonical repository, rename it to `publication-policy.json`, and configure its exclusions, overlay allowlist, synthetic identities, and private terms.

```bash
go run ./cmd/release-gate \
  -source /path/to/private-canonical \
  -distribution /path/to/reviewed-public-checkout \
  -policy /path/to/private-canonical/publication-policy.json \
  -json
```

The command builds the export in a temporary directory and never modifies either checkout.

## Why the overlay is allowlisted

Some public files should intentionally differ: the README, generic command names, synthetic policies, or distribution-specific CI. An overlay makes that difference explicit. Requiring every overlay path in policy prevents a new file from silently becoming public.

## What this demonstrates

- separation between a private canonical system and a public review surface;
- privacy validation before equality validation;
- Git-aware comparison semantics rather than full local permission masks;
- deterministic, non-mutating release checks;
- good/bad fixtures for bypass and compatibility behavior.

Run the complete validation:

```bash
./scripts/validate.sh
```

See [SECURITY.md](SECURITY.md) for reporting and [USAGE.md](USAGE.md) for usage boundaries.

## Portfolio relationship

This repository is the publication-governance layer in the [Jared Silverman GitHub ecosystem](https://github.com/silvermanjared-web/growth-architecture-os/blob/main/docs/ecosystem-map.md). Growth Architecture OS owns the canonical relationship map; the profile repository is the front door; the intelligence, operations, playbook, brand-context, and design-system repositories demonstrate the operating layers. This gate defines how a reviewed public derivative can cross the boundary from private canonical work without exposing private context.

The relationship is architectural, not universal provenance: inclusion in the ecosystem does not mean every public repository is generated from a private source.

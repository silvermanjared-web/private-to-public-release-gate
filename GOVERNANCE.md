# Governance

This repository governs one boundary: private canonical work becoming a reviewed public derivative.

Rules:

1. Every source path has an explicit export decision.
2. Public-only overlay files are allowlisted.
3. Privacy checks run before equality claims.
4. Drift checks use Git-relevant semantics.
5. Findings suppress matched private values.
6. The gate fails closed on ambiguity.
7. The command is non-mutating with respect to source and distribution checkouts.
8. Passing the gate is evidence of this contract, not permission to publish unrelated private material.
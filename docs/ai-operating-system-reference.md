# AI Operating System Reference

This repository implements the publication-boundary pattern from the portfolio's shared AI operating-system architecture.

The canonical reference lives in [Growth Architecture OS](https://github.com/silvermanjared-web/growth-architecture-os/tree/main/04-ai-systems/ai-operating-system-reference).

Local mapping:

- **context** → private canonical source plus publication policy;
- **capability** → build, scan, compare, report;
- **authority** → explicit exclusions and allowlisted public overlay;
- **execution** → temporary, non-mutating export generation;
- **evidence** → privacy findings, drift findings, and pass/fail result.

This is an example of bounded automation at a sensitive boundary: strong enough to execute deterministically, narrow enough to refuse ambiguous publication.
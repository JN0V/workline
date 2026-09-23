# ADR-0002: Name the project workline

- **Status:** accepted
- **Date:** 2026-09-23

## Context

The working name, `assembly-line`, collides with Assemblyline, a well-known
malware-analysis platform — in security, a field the project also touches with
its gates. Names from lean manufacturing were checked on GitHub and the package
registries: `takt` (an AI agent workflow tool, 1.4k stars), `andon` (a quality
system for AI work), `jidoka` (an LLM agent harness) and `pokayoke` (conventions
turned into checks for agents) are taken in this very field. `linewright` is the
name of a company building AI-assisted business workflows.

## Decision

The project, its binary and its folder are named **workline**: `workline`,
`.workline/`, `WORKLINE_*`. It keeps the image of a production line, reads the
same in English and French, and is short to type.

## What the name shares

- Workline, an HR software sold as a service (India); Work Line Connect®, a
  registered Swiss software product linking SAP to manufacturing systems; a
  staffing company in the United States. All are outside developer tooling.
- An npm package `workline` (a tracker of AI work traces, one star). This
  project ships a Go binary, not an npm package.
- Free on PyPI, crates.io and Homebrew on the day of the decision.

## Consequences

- Searches for "workline" alone will show the HR product first; the repository
  description must say what the project is.
- The legal risk is low for an open-source developer tool in another field. If
  the project is ever sold, or bundled into a commercial offer, the name is
  checked by a lawyer first.

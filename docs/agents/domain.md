# Domain Docs

This repository uses a single domain context. These rules describe how the
engineering skills should consume its domain documentation.

## Before exploring, read these

- **`CONTEXT.md`** at the repository root.
- **`docs/adr/`** for architectural decisions relevant to the area being changed.

If these files do not exist, proceed silently. Do not flag their absence or
suggest creating them upfront. The producer skill (`/grill-with-docs`) creates
them lazily when domain terms or architectural decisions are resolved.

## File structure

```text
/
├── CONTEXT.md
├── docs/
│   └── adr/
├── client/
└── server/
```

## Use the glossary's vocabulary

When output names a domain concept in an issue title, refactor proposal,
hypothesis, or test name, use the term defined in `CONTEXT.md`. Do not drift to
synonyms that the glossary explicitly avoids.

If a needed concept is absent from the glossary, reconsider whether the term is
foreign to the project. If it represents a real gap, note it for
`/grill-with-docs`.

## Flag ADR conflicts

If proposed work contradicts an existing ADR, surface the conflict explicitly
rather than silently overriding the decision.

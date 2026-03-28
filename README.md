# GlyphLang Skill

A pi-coding-agent skill for [GlyphLang](https://github.com/GlyphLang/GlyphLang) — the AI-first backend programming language with a spatial assembly substrate.

## What's Inside

```
glyphlang/
├── SKILL.md                      # Main skill reference for pi agents
├── references/
│   └── spatial-assembly.md       # Low-level .glyph opcode reference
├── README.md                     # This file
├── LICENSE                       # Apache 2.0
└── .gitignore
```

## SKILL.md

The core skill definition loaded by pi-coding-agent. Covers:

- **Symbol Reference** — All 13 GlyphLang symbols (`@`, `:`, `$`, `!`, `>`, `+`, `%`, `?`, `*`, `~`, `&`, `#`, `->`)
- **Core Patterns** — CRUD APIs, pattern matching, async/await, WebSockets, generics, auth
- **Type System** — Primitives, arrays, optionals, unions, generics
- **AI Agent Workflow** — `glyph context → validate → codegen` loop
- **CLI Commands** — `run`, `dev`, `compile`, `validate`, `context`, `codegen`
- **Common Mistakes** — Catches `return` vs `>`, `function` vs `!`, etc.

## references/spatial-assembly.md

Full low-level `.glyph` Spatial Assembly reference:

- Complete opcode table (literals, arithmetic, comparison, control flow, metamorphic `M`, biological `S`, I/O)
- Register protocol (lowercase stores, uppercase loads)
- Example programs (arithmetic, conditionals, mitosis, self-modification)
- Spatial grid concepts (Hilbert curves, dimensional layers, Visual Consistency Contract)

## Installation

Copy or symlink into your pi skills directory:

```bash
# Symlink (recommended — stays in sync with repo)
ln -s /path/to/glyphlang ~/.pi/agent/skills/glyphlang

# Or copy
cp -r /path/to/glyphlang ~/.pi/agent/skills/glyphlang
```

## Usage

Once installed, the skill is automatically discovered by pi-coding-agent when tasks involve:

- Writing GlyphLang (`.glyph`) code
- Building AI-optimized APIs or backend services
- Using spatial assembly opcodes
- Working with the GlyphLang CLI
- Integrating with Geometry OS / Ouroboros architecture

## Links

- [GlyphLang GitHub](https://github.com/GlyphLang/GlyphLang) — Language source, compiler, VM
- [GlyphLang VS Code Extension](https://marketplace.visualstudio.com/items?itemName=GlyphLang.GlyphLang)
- [pi-coding-agent](https://github.com/mariozechner/pi-coding-agent) — Agent harness
- [Orchestrator Extensions](EXTENSIONS.md) — AI self-orchestration extensions (SQL, MODEL, Mitosis, VCC)

## License

Apache License 2.0 — see [LICENSE](LICENSE).

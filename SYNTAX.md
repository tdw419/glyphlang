# GlyphLang Orchestrator Syntax

The high-level syntax used in `aipm_core.glyph` and `priority_inject.glyph`.

## Quick Reference

```
# Comments start with \

ROUTE namespace::name -> {
  match args() {
    ("--flag", var) => { body }
    _ => { fallback }
  }
}
```

## Constructs

### Routes
```
ROUTE cli::priority_inject -> { ... }
ROUTE core::orchestrator -> { ... }
```

### Match Arms
```
match args() {
  ("--run", project) => { ... }
  ("--status") => { ... }
  ("--evolve", code) => { ... }
  _ => { FAIL }  # Fallback
}
```

### Variables
```
let count = Q.count("status='pending'")
let prompt = SQL.query("SELECT ...")
```

### Conditionals
```
?autonomous → 
  do_thing()
|
  do_other()

?result.success →
  PRINT "OK"
|
  PRINT "FAIL"
```

### Queue Empty Check
```
?Q∅(queue_var) → 
  PRINT "Empty"
|
  PROCESS queue_var
```

### Loops
```
∞ {
  fetch()
  process()
  sleep(60)
}
```

### SQL Operations
```
SQL.query("SELECT ... WHERE id=?", id) → result
SQL.insert("table", %{field=value})
SQL.update("table", %{field=value}, where_id)
```

### Filesystem
```
FS.read(".loop.control") → cmd
FS.write(".loop.status", %{processed=5})
```

### LLM Invocation
```
MODEL.invoke(prompt, context, model="qwen2.5-coder-7b") → result
```

### Output Analysis
```
OUTPUT.analyze(result.content) → analysis
```

### Self-Modification
```
M[self] ← mutation_code
VCC → 
  ?GREEN → COMMIT
  | RED → REVERT
```

### Mitosis
```
S(0.25, 0.75, ["--status"]) → agent_id
S_WAIT(agent_id) → result
```

### Control
```
PRINT "message {var}"
SLEEP 60
SUCCESS
FAIL
continue
```

## Field Packs

```
%{
  field1=value1,
  field2="string",
  field3=var
}
```

## Interpolation

Variables in strings use `{var}` or `{var.field}`:

```
PRINT "Queue: {pending} pending, {completed} done"
PRINT "Result: {result.content}"
```

## Full Example

```
\ aipm_core.glyph - The AI Orchestrator Brain

ROUTE core::orchestrator -> {
  match args() {
    ("--run", project) => {
      ∞ {
        FS.read(".loop.control") → cmd
        ?cmd.command="pause" → autonomous=false
        ?cmd.command="resume" → autonomous=true
        FS.write(".loop.control", %{})
        
        ?autonomous →
          SQL.query("... WHERE priority<=2") → prompt
        |
          SQL.query("...") → prompt
        
        ?Q∅(prompt) → sleep(60); continue
        
        MODEL.invoke(prompt.prompt, context, model="qwen2.5-coder") → result
        
        ?result.success →
          OUTPUT.analyze(result.content) → analysis
          SQL.update("prompt_queue", %{status="completed"}, prompt.id)
        |
          SQL.update("prompt_queue", %{status="failed"}, prompt.id)
        
        SLEEP 60
      }
    },
    
    ("--status") => {
      let pending = Q.count("status='pending'")
      let completed = Q.count("status='completed'")
      PRINT "📊 {pending} pending, {completed} done"
      SUCCESS
    },
    
    _ => {
      PRINT "Usage: aipm_core.glyph --run | --status"
      FAIL
    }
  }
}
```

## Token Comparison

| Component | Python | GlyphLang | Savings |
|-----------|--------|-----------|---------|
| priority_inject | 231 lines | 47 lines | 80% |
| aipm_core | 800 lines | 60 lines | 87% |
| Full orchestrator | ~2000 tokens | ~300 tokens | 85% |

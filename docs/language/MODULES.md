# Reusable semantic modules

Status: EVT1 R6d next isolated blocker; not implemented

The intended recognizable surface remains:

```concept
module Standard.Generic;
import Standard.Core;
```

One module will own declarations, definitions, templates, and semantic
metadata. There will be no header/source split or textual include model. A
versioned deterministic semantic artifact must carry template representation,
exported signatures, and operation-effect summaries across an acyclic import
graph.

Current `import` admission is profile validation only; it does not resolve
another Concept source or artifact. Consequently R6d's local generic, ABI, and
effect substrate must not be described as a reusable multi-module library yet.
The next implementation step is a semantic module resolver/artifact boundary,
not fixture copying or source concatenation.

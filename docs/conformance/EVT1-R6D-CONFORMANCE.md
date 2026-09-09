# EVT1 R6d generic library substrate

Status: meaningful progression (task end state B)

## Baseline

- Baseline: `8cebcd97c5e4419e0bcb16573bae7b69b60f7cf4`
- Compiler: `concept-evt1-stage0-go`
- R6a: `81b1fc41003130d28b6f2ae4d84e3de71bbdddfa`
- R6b: `79d59cdc38782d4d14a0762f111540347ec23b7d`
- R6c: `8cebcd97c5e4419e0bcb16573bae7b69b60f7cf4`

## Established evidence

Focused compiler tests establish deterministic concrete generic identity and
C layout, nested/cache reuse, integer non-type array extents, runtime
construction, unconstrained function templates, generic layout queries,
generic class methods, bounded recursion failure, structural generic Drop, and
ordinary ref-struct escape rejection.

The hosted seam preserves an external source symbol, validates a bounded ABI
domain, records `ExternalContractEffect` in MIR, compiles with `zig cc
-std=c11`, links a separately implemented host operation, and executes it.

Allocation proof evidence establishes that an external declared `Allocates`
leaf makes a local wrapper's `NoAllocation` assertion Disproven with its call
chain. Existing R6b specimens continue to establish Proven for a closed pure
local graph and Unknown for an opaque operation without a summary. Negative
`NoAllocation` declarations are rejected.

## Isolated blocker

R6d is not complete. Current imports admit names but do not load semantic
dependencies. There is no deterministic module artifact carrying a generic
body or operation-effect summary. This blocks the required cross-module
generic, ownership, provenance, and effect composition proof. Generic type
declaration constraints and module/interface effect contracts should be
completed with that semantic boundary rather than grown as a second local-only
system.

The allocator framework remains unimplemented and must not resume until this
module blocker is removed.

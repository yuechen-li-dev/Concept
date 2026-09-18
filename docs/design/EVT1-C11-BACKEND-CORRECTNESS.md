# EVT1 C11 backend correctness

## Hard invariant

If Concept accepts a closed program for the C11 backend, the generated C must be valid strict C11 and linkable modulo explicitly declared foreign dependencies.

R7d3 enforces this at exact integer constants, fixed-width `int`, checked arithmetic, guarded division and shifts, module-owned case-safe function symbols, pre-emission symbol uniqueness, strict warning-as-error compilation, and independently linked modules.

The backend does not use C as an oracle for signed overflow, division by zero, or invalid shifts. Those operations use the Concept policy in `EVT1-INTEGER-SEMANTICS-DIRECTION.md`.

## Reusable burn-in method

1. Reduce a report to a minimal Concept fixture.
2. Check and inspect emitted C.
3. Compile optimized strict C11 with warnings enabled.
4. Compare with handwritten C only where C semantics are defined.
5. Compare undefined-in-C cases with an independent Go/reference oracle.
6. Repeat generation to prove byte stability.

A proof engine over the wrong expression tree or wrong literal value can confidently prove the wrong property. Arithmetic parsing, symbol identity, and C semantic fidelity therefore precede access-summary derivation.

# Phase 15 C ABI examples

Phase 15 keeps C interop explicit and intentionally small. Concept does not parse
C headers, infer includes, drive a linker, or guess unsupported ABI details.

Supported surface in this phase:

- `extern "C"` declares foreign C functions that Concept code may call.
- `export "C"` emits Concept functions with unmangled C symbol names.
- `[Repr(C)]` marks structs whose field layout is validated for the supported C ABI subset.
- Supported ABI boundary types are `void` returns, `int`, `bool`, `AllocError`, supported raw pointers, `[Repr(C)]` structs by value, and pointers to `[Repr(C)]` structs.

Unsupported and deferred:

- C headers, `#include` generation, automatic linking, or a linker driver.
- C++ ABI or C++ interop.
- Varargs, extern variables, symbol aliasing, and callbacks/function pointers.
- `repr(C)` enums, nested `[Repr(C)]` structs by value, `repr(packed)`, custom alignment, and bitfields.

`[Repr(C)]` is the staged Phase 15 spelling. The keyword spelling `repr(C)` remains reserved and deferred.


## Files

- `extern-abs.concept` demonstrates an outbound `extern "C"` declaration and call to `abs`.
- `export-add.concept` demonstrates an inbound `export "C"` function emitted with an unmangled C name.
- `repr-point-export.concept` demonstrates a staged `[Repr(C)]` struct passed by value across an export boundary.
- `extern-repr-prototype.concept` demonstrates by-value and pointer `[Repr(C)]` extern prototypes.
- `c-abi-boundary-notes.concept` records supported boundary shapes and explicit Phase 15 non-goals.

Phase 15 examples are source examples only. They do not imply header parsing,
include generation, automatic linking, or linker-driver behavior.

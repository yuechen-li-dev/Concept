# EVT1 domain goldens

These are permanent burn-in fixtures. The first natural draft is retained next
to each implementation. Run `go test ./internal/concept -run
TestR7pDomainGoldensNormalAndVerify` for the eight Concept-only domains.
The native companion is a separate project at `tests/goldens/companion`;
run `go run ./cmd/concept build tests/goldens/companion` and
`go run ./cmd/concept test tests/goldens/companion --verify`.

| Domain | Intent and primary surface | Intentional difference from C/C++ |
| --- | --- | --- |
| [Embedded](../../libraries/Golden/Embedded/README.md) | UART command decoder, MMIO, fixed queue, bits, Result | Range `for`, `and`/`not`, explicit `ref`, Euclidean `%` |
| [Civilian aerospace](../../libraries/Golden/Aerospace/README.md) | Commercial flight telemetry transitions, units, fixed history, C ABI value | Explicit scalar-to-quantity trust boundary and typed units |
| [Game](../../libraries/Golden/Game/README.md) | Dense agent store, generational trails, SoA position frame, budgeted ticks, resumable patrol machine | Stable typed IDs and explicit yield, no hidden coroutine heap |
| [HPC](../../libraries/Golden/Hpc/README.md) | Heat stencil and contiguous fixed shapes | `Span` constructed explicitly from backing storage |
| [HFT](../../libraries/Golden/Hft/README.md) | Bounded market queue, C11 atomics, timestamp | Explicit memory order; Unknown synchronization retains atomics |
| [Compiler](../../libraries/Golden/Compiler/README.md) | Postfix parser to dense AST and stable handles | Fixed scalar token stream; payload AST enum; scoped borrows |
| [Native companion](../../tests/goldens/companion/README.md) | Native C++ counter with measured ABI and foreign observer | C++ implementation remains native; semantic claims are explicit |
| [Storage](../../libraries/Golden/Storage/README.md) | Fixed page cache with generational stale-handle rejection | Borrow scope ends before slot removal |
| [CAD](../../libraries/Golden/Cad/README.md) | Unit-typed mesh area and stable vertex/face IDs | Unit literals need a typed binding at call sites |

The goldens intentionally leave application policy local. They do not provide
an ECS, database framework, control framework, or geometry framework.

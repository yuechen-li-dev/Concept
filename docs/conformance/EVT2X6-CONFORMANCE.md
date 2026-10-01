# EVT2x6 conformance

Status: **SUCCESS — EVT2x COMPLETE**, for the bounded scalar pushdown capability.

Compiler: `concept-evt1-stage0-go`, with current embedded innate identity.
Working baseline (merged main): `d576339ec5ff1a8c78805eb9daa3308c94c93eaa`.
Requested clean x5 HEAD: `bca0d0adf242f5b79c36acd922d790be33f5764b`.
Exact x5 addressing commit: `2fb04444d26e3c8d996aa4c74547318bb96b2264`.
The worktree was fast-forwarded to main at the user's request; x5 was its ancestor.

## Acceptance evidence

| Requirement | Real-path evidence |
| --- | --- |
| C oracle | TestEVT2x6COracleBoundaries executes generated C for root complete and root pop, recording every Step boundary and live fields |
| Source push/pop | GenerateLIR emits bounded activation Init/Step, with no source control operation surviving verified LIR |
| Parent continuation | State 1 (Again), then 2 (Done), remains parked under children; C/native compare every live parent field/state |
| Child initialization | Validated ordinary expressions, private fields, state zero, continuation, tag, then depth; capacity precedes child storage and continuation |
| Initializer scope | A parent local `value=99` cannot shadow shared bindings during construction; focused test pins shared load |
| Depth safety | Completed check, then positive live depth before subtraction, then index bounds before each address |
| Dynamic dispatch | Actual top tag selects a machine-specific state/field layout; invalid depth/tag/state traps before reinterpretation |
| Typed destruction | Exhaustive closed tag chain enters scalar typed destroy CFG regions before reduced-depth publication; no pop pseudo-op |
| Child yield | Count persists 4 -> 5 -> 6; yield returns, preserving parent and child state; next Step starts the same state |
| Child transition | State 0 -> 1 changes only child state; depth and parked parent remain unchanged |
| Child completion | Neutral complete and pop remove the child and stop the Step; parent executes next Step without copying/reconstruction |
| Root completion/pop | Depth zero, completed true; subsequent Step returns Completed with no byte mutation. Preserving C root-pop semantics was authorized by the user |
| Repeated child | Second activation starts at count 4 after the first was destroyed at count 6 |
| Nested depth | Parent -> Child -> GrandChild reaches depth three through generic source lowering |
| Distinct frame types | Parent/Child int at frame offset 4; GrandChild bool at 4 and int at 8; all live fields compare with C |
| Failure invariants | Overflow, initializer overflow, incomplete depth zero, Capacity+1, root/nested corrupt tag/state trap with byte-identical complete frame and sentinel snapshots |
| Sentinels | Both 108-byte stride-12 source frame and 140-byte nested frame preserve 16 bytes on each side |
| Multiple instances | Independent native frames with different depths/shared values execute interleaved, each matching its own C instance |
| Normal/Verify | C oracle and Concept-backend hosts under both policies match the same conservative guarded native LIR path |
| Determinism | 100 complete LIR/layout generations, LIR/MachineIR text and CMIR bytes; 100 physical-assignment/native-byte comparisons; 100 C/native logical trace runs |
| NoAllocation | Verified generated Step has no call/heap operation and fixed caller storage. Formal generated-Step proof is unavailable; backend facts and NoAllocation(LayoutLoweredBlocks) pass |
| Bounded | Capacity 8; Parent/Child max frame 8/align 4, stride 12, automata 108/4; nested max frame 12/align 4, stride 16, automata 140/4 |
| Backend | CMIRAMD1 and Go MachineIR lowerer/verifier unchanged; no automata opcode, new encoder form, spill, assembler, heap fallback, scheduler, or async path |

## Native traces

Result encoding: Active=0, Yielded=1, Completed=2. C's void Step API is observed
through its existing yield marker and actual typed instance. No machine semantics
are implemented by the oracle harness. Differential checks include every live
frame, not merely the trace projection or final result.

Source Parent/Child fixture uses actual stride-12 multiply + scale-one LEA:

| Step | Result | Depth | Shared value |
| --- | --- | --- | --- |
| 0 | Active | 2 | 0 |
| 1 | Yielded | 2 | 5 |
| 2 | Active | 1 | 11 |
| 3 | Completed | 0 | 18 |
| 4 | Completed | 0 | 18 |

Nested fixture columns: Step, result, depth, top tag, top state, parent.preserved,
top Child.count if live, shared value. Inactive root bytes stay 7; reading them in
the test is not a language-level use of a destroyed frame.

```text
0   0 2 1 0 7  4  0
1   1 2 1 0 7  5  0
2   0 2 1 1 7  6  0
3   0 3 2 0 7 -1  0
4   0 2 1 2 7  6 10
5   0 1 0 1 7 -1 16
6   0 2 1 0 7  4 16
7   1 2 1 0 7  5 16
8   0 2 1 1 7  6 16
9   0 3 2 0 7 -1 16
10  0 2 1 2 7  6 26
11  0 1 0 2 7 -1 32
12  2 0 -1 -1 7 -1 39
13  2 0 -1 -1 7 -1 39
```

## Verification and shared backend repair

17 malformed-LIR mutations are rejected: missing positive-depth guard, early
depth publication, wrong tag, missing persistent initializer, wrong initial
state, out-of-slot write, omitted continuation, missing capacity guard, reduced
depth before destroy, wrong destruction type, missing tag arm, missing pop
index/depth guard, post-destroy frame use, parent access through child layout,
missing state dispatch, incoherent child type/tag, and removed push provenance.

The source Step exposed CapacityExceeded (old liveness bound: 16 blocks / 32
registers), then artificial RegisterExhausted because split check continuations
were appended far from their defining blocks. Both repairs are written in
Concept: fixed bitsets for 128 blocks / 512 registers, plus grouping/remapping
split blocks and instruction ranges before intervals. The pool and allocation
strategy are unchanged; spills remain explicit RegisterExhausted. Instruction
forms and CMIRAMD1 are unchanged. Nested Step has 267 virtual registers and 106
MachineIR blocks. Concept tests cover register 511 through 128 blocks, capacity
failure, branch/instruction remapping, idempotence, and NoAllocation of layout.

## Commands and qualification limits

```console
go test ./internal/concept -run "Test(EVT2|MachineStack|SemanticCorpus|Innate|AutomataTransitions)" -count=1
go test ./... -count=1
go vet ./...
go run ./cmd/concept test libraries/Standard
go run ./cmd/concept test libraries/Standard --verify
go run ./cmd/concept test libraries/DragonGod
go run ./cmd/concept test libraries/DragonGod --verify
go run ./cmd/concept test libraries/Golden
go run ./cmd/concept test libraries/Golden --verify
zig build test
# From legacy/poc3-zig:
zig build test
go run ./cmd/concept lir language/evt2/machines/parent_child.concept
go run ./cmd/concept machineir language/evt2/machines/parent_child.concept
go run ./cmd/concept amd64 language/evt2/machines/parent_child.concept
go run ./cmd/concept lir language/evt2/machines/pushdown.concept
go run ./cmd/concept machineir language/evt2/machines/pushdown.concept
go run ./cmd/concept amd64 language/evt2/machines/pushdown.concept
```

Windows AMD64 executable-memory execution is qualified; Linux/macOS native
execution is unverified. Native Verify uses one conservative guarded LIR policy;
C oracle/backend Verify hosts are exercised. Owned activations/observable Drop,
typed completion payloads, input reactions, guarded transition-match, and
pushdown terminal settling retain explicit boundaries. Legacy Main/InstanceDecl
wrappers need later call lowering; the machine declarations and CLI fixtures are
qualified. EVT2e is not begun.

Finite-machine tests, EVT2d Add/Max/Sum4/CheckedIndex/StoreIndex/Choose/Early,
root Init, standalone x5 addressing, and semantic/innate corpus tests are retained.
Final counts, commits, and worktree status are recorded in EVT2X6-CONVERGENCE.md.

Final gates: full Go suite and vet pass; focused EVT2/machine/semantic/innate
corpus pass; Standard Normal/Verify 42 each, DragonGod 23 each, Golden 130 each;
root and legacy Zig suites pass. The final x6 acceptance run passes after the
complete-metadata and physical-assignment determinism checks were added.

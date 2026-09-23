# EVT1 R7f3 collector collections: honest stop

Baseline `3e830976f5d0d43a9e63bb332c1a19d2860a9b60` is R7f2. R7f1 is
`58dae662d4559b2c0ea6843767e36a54dfbd706d`; R7e is
`b1747d7d3ffba7510aff6484c6863f70c214cf6e`. Compiler ID:
`concept-evt1-stage0-go`. The initial worktree was clean.

No `Standard.Collection` package is shipped. The attempted homogeneous,
allocator-backed representation used fixed metadata arrays and an
`Option<Storage<T>><array>[Capacity]` authority table. An empty table can be
constructed, but placing a freshly bound, initialized `Storage<T>` into an
indexed slot with `slots[index] = move candidate` reports `CV4133: assignment
copies non-copyable type Option<Storage<T>>`. The minimal reproducer is
`TestR7f3IndexedStorageAuthorityMoveRemainsRejected`. Relaxing that check
without a general rule for replacing or taking non-copyable indexed elements
would permit authority duplication or loss. Encoding live objects only as raw
regions would bypass the initialized typed-storage authority and its
Destroy-before-Release law. Neither is an acceptable collector implementation.

One general syntax gap was closed: an applied type can now be the element of a
fixed array, for example `Box<int><array>[N]`. This is checked by
`TestR7f3AppliedTypeFixedArraySyntaxInStrictC11`, which instantiates an open
non-type extent and executes strict C11. The change is independent of any
collector name or runtime policy. An additional observed code-generation gap
is that an `Option<int><array>[4]` local generated a C array before emitting
the `concept_option_int` type declaration. That path is not claimed working.

The next prerequisite is a general, sound move-into/take-from indexed storage
operation for non-copyable elements, with explicit behavior for replacing a
live value and correct Drop. The applied enum-array C declaration ordering
also needs correction if optional slots are used. This is broader language
and backend work than the bounded collector retry; no unsafe bypass or
collector-specific compiler branch was added.

Validation after removal of the incomplete library: `go test ./...`,
`go vet ./...`, both `zig build test` lanes, and the focused R7f3 tests pass.
An Oct executable built from the adjacent checkout ran `make BurnIn`,
`BuildStandard`, `TestStandard` (3 facts), and `TestDragonGod` (21 facts)
through `Make.oct`. The final Standard package graph matches the baseline
content hash. A production-source audit found no collector-specific compiler,
MIR, planner, runtime, Standard, or DragonGod branches.

Consequently there is no claimed mark/sweep, cycle collection, root table,
stable handle, borrow/Collect effect, NoAllocation proof, artifact-only
consumer, scheduler coordination, or collector determinism result. R7f1,
R7f2, R7e, Standard.Memory, table/ellipsis, and DragonGod behavior remain
covered by the existing full regression gates. R7g reflection remains
independent and was not started.

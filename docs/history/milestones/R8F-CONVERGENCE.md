# R8f convergence

The starting worktree was clean at `30262299fc78d10a9c8367636e48cd457e1974e5`.
That merge commit has `83ff006db657bf23bf6b86d4259a292b7236ef98` as its
first parent and no compiler content delta. Compiler ID:
`concept-evt1-stage0-go`.

Baseline passed `go test ./... -count=1`, `go vet ./...`, root and legacy Zig
tests, BurnIn, Standard Normal/Verify (35 facts each), and DragonGod
Normal/Verify (23 facts each). The full Go test includes the R7p goldens, R7q
Frame, R8a-R8e3, and semantic corpus.

R8f first brought five new domains into the real native test runner. The
mechanics case rejected implicit float-to-double tensor literals. The game
case required each yield/transition to terminate a machine step. The HPC case
exposed missing concrete span declarations and dead closed-template branches
in C. The async case exposed return-bearing branch lowering in void step
functions and MustUse annotation confusion in test discovery. Each compiler
issue was repaired at its local authority and pinned by a focused test.

Adding two native benchmarks exposed an R7p aggregate-test accounting error:
the manifest includes benchmarks, while `Passed` counts only facts. The
assertion now checks `Passed + Benchmarks` against the manifest. The focused
R7p domain test passes in Normal and Verify with 37 facts and two benchmarks.

The full `go test ./... -count=1` passed after the five domains and compiler
fixes were introduced (204.225 s for `internal/concept`). A final rerun also
passed after adding the compiler golden's generated-pass semantic concept and
its authored-pass rejection. `go vet ./...`, both Zig suites, BurnIn, the complete
Golden tree (37 facts and two benchmarks), Standard Normal/Verify (35 each),
and DragonGod Normal/Verify (23 each) passed. The new native facts agree in
Normal and Verify. The representative 100-run artifact/C identity test,
artifact-only consumers, and strict GCC/Clang C11 test pass. `concept
generated` and `concept explain` were run against built semantic artifacts.
Project lint passes with `CONCEPT_MODULE_ROOTS` set to the absolute
`libraries` directory; individual format checks pass for all 12 new
sources/manifest files.

The final compiler example requires `CheckedOperandPass<declaration
OperandCount>`, which proves generated provenance and `NoAllocation`. An
authored replacement fails with `CONCEPT_ASSERT_DISPROVEN`. With a fixed
artifact root, `concept generated` and `concept explain` each produced
identical output across 100 invocations; project lint and the compiler source
format check did the same.

The new R8f tree leaves every R7p golden and the R7q Frame unchanged. The
remaining language-design opportunity is open-type arithmetic bounds; the
golden uses existing required-operation closure instead of adding syntax.

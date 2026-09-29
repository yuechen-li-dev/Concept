# R7q convergence log

1. Confirmed clean R7p HEAD `43a6af3fd566020a480231187d1925b4121c6b17`
   and compiler ID `concept-evt1-stage0-go`.
2. Drafted a coordinate-frame workload from `world_i = basis_ij local_j`.
   The first draft used raw literals for a unit-typed tensor and received
   `CV4227`. The quantity guide explained the explicit measurement boundary
   `AssumeQuantity<float<m>>`; after applying it, the program checked.
3. Replaced the identity basis with a quarter turn. The independent geometry
   fact passes in Normal and Verify. The generated C uses fixed 3-element
   backing storage and nested contraction loops, but retains runtime shape
   fields and extent guards. Clang and GCC both accept the generated pair
   under strict C11. It has no tensor heap or runtime reflection.
4. A deliberately wrong dimensional contraction receives `CV4615`, including
   the inferred `float<m^2>` and declared `float<m>` types.
5. The required HPC width probe receives `CV4102` for `float<32>`. The frozen
   quantity guide explicitly reserves `float<K>` for Kelvin, and the compiler
   has one base floating representation. This is a missing semantic capability,
   not a syntax discovery issue or a local implementation bug.
6. Baseline `go test ./...`, `go vet ./...`, both Zig test suites, BurnIn,
   Standard Normal/Verify, DragonGod Normal/Verify, the R7p golden test, and
   the native C++ companion Normal/Verify pass. The full Go test includes the
   frozen EVT1 semantic corpus.

The geometry case remains a small, working independent example. The probe
sources are retained as text so test discovery does not mistake intentional
invalid programs for buildable modules. No existing R7p source was changed.

R7q stopped here because implementing width-parameterized floating point
would reopen frozen type, arithmetic, artifact, and backend rules. The minimal
proposal and compatibility consequence are in
[EVT1-R7Q-CONFORMANCE.md](EVT1-R7Q-CONFORMANCE.md).

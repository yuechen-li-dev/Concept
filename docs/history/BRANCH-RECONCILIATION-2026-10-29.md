# Main branch reconciliation, 2026-10-29

The primary checkout now uses `main`. Its local starting point was `8b129ad`,
remote main was `4e96e5c`, and the completed task checkout was `8cf2692`.
Main was fast-forwarded through the remote baseline and all four h2 commits.
EVT2h2 remains explicitly PARTIAL: checked payload LIR only, without native
payload execution. Consolidating branches does not change that status.

## Divergent history reconciled

- `59d35e2` records the original EVT2f lineage through `6f03274` as merged.
  All 33 changed paths were already included by checkpoint `aef7824`, which is
  an ancestor of main: 32 files match exactly after CRLF/LF normalization, and
  AMD64.concept has all the old content plus 189 added lines and no removals.
  Later changes remain selected. This is a history-only `ours` merge, justified
  by that content audit, not a replay of obsolete code.
- `7395e59` records `75e4dc7` as merged. Its stable patch ID exactly matches
  already-integrated `a657336`:
  `44276cbe7dcde4d443efc3b94c44a21d829fda5c`.
- Both merge results retain the exact `8cf2692` tree:
  `e7dde591d2dfd176cad40dd67cee928ea6bd77b9`.
  Only this audit and the branch-lifecycle guidance in AGENTS.md are added
  after that equivalence check. No compiler/library/fixture bytes are changed.

## Verification and preservation

Quick passed on reconciled main in 26.16 seconds. Focused h2, frozen h1
transport/source boundaries and R9c authority checks passed in 3.574 seconds;
vet and diff checks passed. The runtime code is identical to the already
qualified h2 checkpoint (`09b3d50` plus documentation at `8cf2692`), whose full
qualification passed in 1419.61 seconds. Full native compilation was therefore
not repeated for these history-only merges and documentation changes.

Six historical worktree folders are retained at their original commits with
HEAD detached, freeing their obsolete branch refs without deleting files.
The R7l2 checkout initially reported 119 modified paths, but every file was
byte-for-byte identical to its HEAD blob. Those bytes were additionally saved
under ignored `artifacts/branch-reconciliation/r7l2-preserved/` and verified
unchanged after detaching. Its status/index reporting was not repaired by
rewriting files or changing shared Git configuration.

All branch tips below are verified reachable from main before cleanup. Their
commits remain in main's history and can be restored with
`git branch <name> <tip>`. The cleanup removes 30 local task refs and 17 remaining
remote task refs, after publishing integrated main. Nine already-deleted remote
refs were pruned during fetch. No force push, forced branch deletion or worktree
folder deletion is used.

## Reconciled branch inventory

| Branch | Local tip | Remote tip |
| --- | --- | --- |
| codex/add-amd64-machineir-lowering | a6aa92d1ccd01e9e45286ba3dcaa2085e1755e3d | a6aa92d1ccd01e9e45286ba3dcaa2085e1755e3d |
| codex/add-ast-and-parser-infrastructure | — | c3cde344b0c4401e40e76acf4f99a5e5155efd4b |
| codex/add-mirtolir-lowering-mvp | 23a5c0f836aa5ed658601a1bac48b4a6923ddb9d | — |
| codex/add-phase-6-unsafe-and-ownership-design-document | — | b4f034d4c42aacec3bf3d738f5b8d65cae58e1fa |
| codex/add-support-for-semantic-generic-functions | — | a6a3593a0cf31f30959ff47eb5c0ef8cce373c1c |
| codex/concept-test-workflow | bc84b511f307391c7953b73ffd0b8a5ec0c3dc30 | bc84b511f307391c7953b73ffd0b8a5ec0c3dc30 |
| codex/evt2c-machineir | f5137ee5bab5db7e89da1bc5740eeca7c1e1d594 | — |
| codex/evt2d-concept-amd64 | 35b539b88ebe081e3814c2e02a574821f569384c | 35b539b88ebe081e3814c2e02a574821f569384c |
| codex/evt2e-cathedral-calls | 22cdbf5934bf7096442bdbf23da786ce0ee934f1 | — |
| codex/evt2f-scalar-input | 9b1b9ea25d5f017b9146db676d02f914d348601f | — |
| codex/evt2f2-cmiramd4 | 2f8443865aab1f9d1407735426c32f27751efb98 | — |
| codex/evt2f3-xmm-planning | 6f03274541fcb9a6426c99b04dfbae3e0bf21233 | — |
| codex/evt2f4-latest | 9d57d649b0edca082c0af6f15b9827a05ef363e1 | — |
| codex/evt2f4-scalar-sse2 | 6f03274541fcb9a6426c99b04dfbae3e0bf21233 | — |
| codex/evt2g-checked-aggregate-input | e51c184bf51e6073652073293ed5587e26546ba6 | e51c184bf51e6073652073293ed5587e26546ba6 |
| codex/evt2g2-cmiramd5 | 7ccb968ab79d4ad700db7afa396970aa78868d06 | 7ccb968ab79d4ad700db7afa396970aa78868d06 |
| codex/evt2g3-native-objects | 2a8b6487adbc72c800120e0340db27f0aa5ee2ac | 2a8b6487adbc72c800120e0340db27f0aa5ee2ac |
| codex/evt2g3b-native-references | e7c10a81324abb7511ef4a42d49d27e31f543f4b | e7c10a81324abb7511ef4a42d49d27e31f543f4b |
| codex/evt2g4-aggregate-abi | bcfe25c52627127873242e91a077a14dbe7e186b | bcfe25c52627127873242e91a077a14dbe7e186b |
| codex/evt2g5-native-aggregate-abi | 535910e486b0f20f6ec1c6e6e7bee931d359b61a | 535910e486b0f20f6ec1c6e6e7bee931d359b61a |
| codex/evt2h1-native-enums | c250308f89bc7869b2c0a99efd3f846856c6e6d8 | — |
| codex/evt2h2-native-payload-enums | 8cf269269b67ebfb518cf126902dcb04daa55285 | — |
| codex/evt2x-native-automata | 58d899927ef512b52f1a5192efb07376f0db43b3 | — |
| codex/implement-dg7-dragongod-actuation-subsystem-v0 | — | c1db026b4ff29a25c664521810874a9dae160dd1 |
| codex/implement-import-declaration-parser-and-diagnostics | — | bb22db161d86e2004d8711531e5c36b0ea15b579 |
| codex/parse-payload-enums-in-parser | — | 96d64b0e3b22b40e931bef90652ec3ac9fbf55e4 |
| codex/r7k-hardware-memory | a51a8a6dc9537420fa7caa1f9cdb09c280185c15 | — |
| codex/r7l2-machine-operations | 75e4dc7e10008089fea8ec7267d3b0d3db6933f0 | — |
| codex/r8e3-source-format | 83ff006db657bf23bf6b86d4259a292b7236ef98 | 83ff006db657bf23bf6b86d4259a292b7236ef98 |
| codex/r9a-stage0-closure | 4474af15f95922d8362a132bd80093a6ae0105dc | — |
| codex/r9b-semantic-vocabulary | 9c0c1cb5d6797c126df50d97f6e141b3a1dcd767 | — |
| codex/r9b2-typed-verdicts | b2e3388bd835687b8835d5358fb5732ccc921433 | — |
| codex/r9b3-semantic-vocabulary | 3cb83bc4e4d9ac96fc86b151380a2c129cc60ca8 | — |
| codex/r9c-cathedral-transition | a85747a2ff061848e06ecbc10e68d88710cf278c | a85747a2ff061848e06ecbc10e68d88710cf278c |
| codex/r9d-prefer-match | 9616fcefb98f2721009068a89c68d90a7321fabb | — |
| codex/test-build-reuse | aaa4cd57cceed05737f1998188f5b57e84c2be6a | — |

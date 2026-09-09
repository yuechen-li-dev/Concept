# EVT1 test prophecy direction

Prophecy is a doom test: the declared behavior is a catastrophic or abnormal
termination path that cannot safely execute in the host test process. R6a
therefore compiles a static ordinary-Concept harness and launches it as a
bounded child. Any abnormal exit fulfills a prophecy; a normal return or
timeout fails. Exact panic/signal/exception matching is deferred.

`[[foretold]]` does not change what fulfills the prophecy. It arms enhanced
evidence retention: test identity and source, timing, normalized and raw
termination details, exact stdout/stderr, panic text when available, artifact
paths and hashes, build/compiler/target identities, and up to the last 16
`Foretell.Checkpoint` records. Missing platform detail is omitted rather than
fabricated.

Evidence is written to the deterministic `latest` directory for that test.
The structured result remains the authority and the text files preserve exact
streams for crash archaeology. The mechanism is test tooling only: ordinary
function bodies, terminal panic behavior, MIR semantics, and generated C11
remain unchanged, and no generalized tracing or crash debugger is introduced.

# R4l invalid evidence

R4l adds compiler architecture rather than source semantics. Valid Concept
programs therefore do not become invalid merely because an optimization fact
is Unknown. Malformed-plan, stale-MIR, missing-guard, invalid-witness,
incomplete-cleanup, unsupported-target, and absent-fact cases are constructed
and rejected directly in `internal/concept/planner_test.go`.

# Civilian flight telemetry

A commercial flight telemetry controller validates native samples, applies
unit-typed envelope checks, records a fixed history, and moves through
preflight, climb, cruise, descent, and fault states. Its first draft is
preserved in `first-draft.concept.txt`. Normal and Verify use
`flight_telemetry.concept_test`.

Friction: C-style loop and nominal `unit` declaration were natural instincts.
The actual spelling is a range loop and numeric representation with unit, such
as `float<m>`. The native ABI uses raw floats, so `AssumeQuantity<T>` is the
explicit trusted interpretation point. Familiar: records, enums, bounded
buffers. New: quantity types and `Result`. Advanced: `NoAllocation` proof.
`NativeSample` has local `repr(C)` eligibility. An artifact-only
`CAbiValue<NativeSample>` claim remains Unknown until a native toolchain probe
measures the selected external struct. The separate native companion golden
demonstrates measured ABI authority and Verify's foreign observer.

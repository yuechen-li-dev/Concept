# Standard library

`libraries/Standard` is the production Standard package. Its
`manifest.concept` credits `CODEX`, has no dependencies, and is semantically
checked against the ordinary schema in `Standard.Build.Metadata`.

R7a promotes the unchanged R6p Standard.Memory sources here. The public module
identities, allocator concepts, typed allocation owners, region geometry,
effects, strict-C11 behavior, and deterministic module artifacts are preserved.
The package builds independently with `go run ./cmd/concept package build
Standard` and tests with the corresponding `package test Standard` command.

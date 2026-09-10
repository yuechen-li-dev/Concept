# External C ABI declarations

Status: EVT1 R6j ABI seam plus separate foreign semantic authority implemented

An external C operation is declared without a body:

```concept
extern "C"
byte* ConceptHostAllocate(usize size);
```

The source operation name is the linker symbol. The Generic C11 backend emits
an ordinary external prototype and call; it does not add a runtime symbol
registry or hard-code libc allocation.

An extern declaration remains ABI-only. R6j adds a separate, explicitly bound
foreign semantic declaration for reviewed effects and result storage facts.
Its current source spelling is provisional; see
`docs/interop/FOREIGN-CONTRACTS.md`. An extern without such a declaration stays
Unknown for semantic proofs, and its pointer is not trusted by `bind<T>`.

The initial ABI domain is deliberately small: Core scalar builtins, enums, and
single pointers to builtin storage. References, `dyn`, async values, callbacks,
arrays, applied types, and aggregate values are rejected with
`EXTERN_C_ABI_TYPE_INVALID`. A link-name annotation and stable aggregate ABI are
deferred.

# External C ABI declarations

Status: EVT1 R6d hosted seam implemented

An external C operation is declared without a body:

```concept
extern "C"
byte* ConceptHostAllocate(usize size);
```

The source operation name is the linker symbol. The Generic C11 backend emits
an ordinary external prototype and call; it does not add a runtime symbol
registry or hard-code libc allocation.

The initial ABI domain is deliberately small: Core scalar builtins, enums, and
single pointers to builtin storage. References, `dyn`, async values, callbacks,
arrays, applied types, and aggregate values are rejected with
`EXTERN_C_ABI_TYPE_INVALID`. A link-name annotation and stable aggregate ABI are
deferred.

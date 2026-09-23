# Standard.Collection status

`Standard.Collection` is not available in R7f3. The ordinary-library
collector attempt stopped at the non-copyable indexed storage-authority
boundary described in
[EVT1-R7F3-CONFORMANCE.md](../conformance/EVT1-R7F3-CONFORMANCE.md).

No application should infer collector behavior from the experimental syntax
support for fixed arrays of applied types. `Standard.Memory` remains the
supported explicit allocation and typed-storage library.

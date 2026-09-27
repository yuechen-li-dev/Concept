# EVT1 R7k MMIO direction

The address/region/type split from R6g remains authoritative. `DeviceMemory`
is a nominal address-space tag. The low-level `MmioLoad<T>` and
`MmioStore<T>` intrinsics are explicit effects over device addresses. They do
not introduce a second pointer family or an ambient volatile qualifier.

The shared semantic path admits four unsigned widths, records `mmio_read` and
`mmio_write` operations in MIR, requires `RetainOrderedVolatileAccess` in the
planner, emits one strict-C11 volatile access per operation, and derives
`HardwareRead`/`HardwareWrite` facts through calls. Module artifacts carry
closed hardware summaries and scalar-backed bits declarations. A missing or
opaque summary remains Unknown.

The C translation's compile-time macro seam allows the hosted simulator to
record address, width, value, and sequence without changing production
semantics. Default macros are width-specific volatile accesses. They preserve
the order of explicit MMIO full expressions; they do not assert CPU memory
ordering. Future LIR and MachineIR backends must retain that same observable
operation contract. Architecture-specific device fences belong with later
target support.

DragonGod's UART is ordinary library code over the same intrinsics. DMA,
cacheability, page mapping, PCI BAR discovery, port I/O, interrupts, and inline
assembly remain separate milestones.

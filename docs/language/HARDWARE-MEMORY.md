# Hardware memory (R7k)

MMIO is an observable hardware operation. It is not a volatile-qualified ordinary object.

`Standard.Hardware.Mmio` provides the nominal `DeviceMemory` address-space tag.
`Address<DeviceMemory>` has the same affine byte arithmetic as other addresses;
there is no implicit conversion from `Address<SystemMemory>` or another space.
`MemoryRegion<DeviceMemory>` remains ordinary bounded geometry. Supplying a
device address is a trusted platform/foreign boundary. `AddressFromBits` alone
does not establish region provenance or make a `SystemMemory` storage binding
safe. R7k's low-level primitive does not perform a region-bounds check.
The existing foreign-storage authority can establish a bounded
`MemoryRegion<DeviceMemory>` from a live platform lease: declare a foreign
`ExternalStorage<DeviceMemory>` contract, then call
`EstablishExternalRegion<DeviceMemory>` with that lease. The
`TestDeviceRegionUsesExistingForeignAuthority` fixture validates this exact
path. Region-checked register descriptors remain a higher-level library
option; the primitive's address is explicitly trusted.

`MmioLoad<T>(address)` and `MmioStore<T>(address, value)` accept only
`Address<DeviceMemory>` and `T` equal to `uint8`, `uint16`, `uint32`, or
`uint64`. Legacy `uint` is the 32-bit scalar spelling and is also accepted.
Each call performs exactly one native-width device transaction. A load with an
unused result still happens. Repeated reads and writes remain separate.
The primitive does not swap byte order. Misaligned constant addresses are
rejected when visible through an immutable literal `AddressFromBits` chain;
unknown alignment is left to the platform/device contract.

R7k preserves explicit MMIO transaction order. CPU/device memory-barrier
semantics beyond that are separate explicit operations. The C11 backend emits
one width-specific volatile access for each primitive and never adds volatile
to a source type. The default macros implement the access; a hosted test can
replace them at C compile time with a transaction recorder. The macros do not
claim a CPU fence or device cacheability policy.

MMIO differs from C11 atomics, which synchronize threads over ordinary
memory. A loaded register value is ordinary data; inspecting its bits cannot
trigger another MMIO read. `HardwareRead` and `HardwareWrite` are compiler
facts over operations and transitive call graphs. They travel in semantic
module artifacts. `NoAllocation` continues to derive independently.

DMA, cacheability, page mapping, port I/O, interrupts, and inline assembly
are not part of R7k. Architecture-specific barriers and native backends must
preserve the same observable MMIO operations when those capabilities arrive.

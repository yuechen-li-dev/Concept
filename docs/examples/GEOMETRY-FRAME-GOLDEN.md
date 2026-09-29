# EVT1 differentiator goldens: R7q stopped at a frozen boundary

R7p's [migration goldens](DOMAIN-GOLDENS.md) remain the permanent layer for
ordinary industry workloads. R7q began a separate layer under
`libraries/Golden/Differentiators` to test whether advanced Concept semantics
change the program's structure.

| Mechanism | New example | Qualification |
| --- | --- | --- |
| Tensor shape, Einstein contraction, and unit-typed elements | [Frame](../../libraries/Golden/Differentiators/Geometry/Frame.concept) | Quarter-turn result passes Normal and Verify; dimension mismatch rejects with `CV4615` |
| Width-parameterized `float<K>` | [Precision probe](../../libraries/Golden/Differentiators/Hpc/precision-probe.concept.txt) | Blocked: `K` denotes Kelvin, `float<32>` receives `CV4102` |

## Feature discovery and first-draft friction

| Feature attempted | Problem and first attempt | Documentation and result | Classification |
| --- | --- | --- | --- |
| Unit-typed tensor literal | Measured displacement from raw `1.0`, `2.0`, `3.0` | `CV4227`; the quantity guide led to an explicit semantic interpretation at the measurement boundary, now spelled `interpret value as float<m>` | Documented intentional boundary |
| Einstein contraction with units | `world[i] = basis[i, j] * local[j]` | Checks and runs; a wrong unit result receives `CV4615` | Working composition |
| Precision width | `float<32>` and `float<64>` for one HPC algorithm | Quantity guide defines `K` as Kelvin; compiler rejects `float<32>` with `CV4102` | Post-EVT1 feature proposal |

The frame source directly states the indexed mathematical relation; a typical
C++ implementation would use loops or an external tensor library, then a
separate units library to type the measured vector. Its generated C keeps
fixed storage and explicit loops. This is a bounded example, not a completed
R7q geometry or HPC qualification. The remaining proposed differentiator
rows are intentionally absent until executable examples support them.

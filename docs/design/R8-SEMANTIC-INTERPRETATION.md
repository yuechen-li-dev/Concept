# R8d semantic interpretation

`interpret expression as T` marks the point where a programmer attaches
externally supplied quantity meaning to a scalar. The source may be a foreign
function result, a decoded file field, a protocol field, or another value whose
unit is specified outside Concept. Ordinary code may use the syntax; it is the
explicit declaration at the site that matters.

The accepted R8d operation is an unqualified numeric scalar to a quantity
with the **same scalar representation**. `float` to `float<m>`, `double` to
`double<Pa>`, and supported integer scalars to their same-representation
quantity types are examples. The stored value is unchanged. The target unit
is attached exactly as written: interpreting 1200 as `float<mm>` means 1200 mm.
There is no runtime unit object, allocation, or scale arithmetic.

`as` performs language-defined numeric conversion and exact same-dimension
unit-scale conversion. `interpret` supplies meaning that cannot be derived
from the value. `Magnitude` extracts the scalar stored in a quantity's current
unit. Thus:

```concept
float raw = 1200.0;
float<mm> distance = interpret raw as float<mm>;
float<m> meters = distance as float<m>;
float rawMillimeters = Magnitude(distance);
double<m> widened = interpret (raw as double) as double<m>;
```

The first operation is semantic attachment, the second scales by 1/1000, the
third restores 1200, and the fourth converts representation before attaching
meaning. `static_cast<T>(x)` remains an alias of `x as T`.

`interpret` cannot strip a quantity, convert between quantities, change a
dimension, reinterpret integer bits as a float, alter const or reference
authority, construct an object from storage, or change address space. Those
operations use their existing explicit authorities or reject. The legacy
`AssumeQuantity<T>(value)` primitive remains accepted for source compatibility;
new authored code uses `interpret`.

MIR retains `interpret_scalar_semantics` with source/target types, normalized
dimension, exact scale, source span, and `ExplicitInterpretation` provenance.
The semantic artifact keeps inspectable interpretation sites, and `concept
explain file:line --verbose` can inspect a site. The explanation proves only
that the boundary was explicitly declared and type checked. It does not prove
that an external producer actually followed its documented unit convention.

The parser gives the interpretation operand unary precedence. Parenthesize a
compound source: `interpret (x + y) as float<m>`. A later cast is clear as
`(interpret raw as float<mm>) as float<m>`.

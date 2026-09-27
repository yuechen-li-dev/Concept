# Handwritten C hot-path shapes

`hot_shapes.c` is the intentionally plain C11 structural comparison for the
HPC, HFT, game, and compiler goldens. It has no framework and no timing
harness. Compile with `clang -std=c11 -pedantic-errors -Wall -Wextra -Werror
-fsyntax-only hot_shapes.c` or the GCC equivalent.

The generated Concept C keeps inline arrays and atomics but carries explicit
range state, overflow helpers, and repeated index guards. The handwritten
shapes show the lower-bound code shape for a future backend study. This
comparison does not claim benchmark parity or motivate an EVT1 semantic
change.

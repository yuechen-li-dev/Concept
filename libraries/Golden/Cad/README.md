# CAD mesh kernel

The mesh stores value vertices and faces under typed dense IDs and computes a
unit-typed signed area, centroid, bounds, and degeneracy count.
`mesh.concept_test` exercises orientation and inline
traversal. The first C++-instinct draft is preserved in
`first-draft.concept.txt`.

Friction: `auto`, pointer arithmetic, and vector operators gave way to
explicit values and IDs. Unit literals require a typed local before a call.
Familiar: geometry formula and fixed-capacity arrays. New: typed IDs and
quantities. Advanced: `NoAllocation` proof.
The centroid revealed a unit-propagation bug for `length / 3.0`; scaling by a
dimensionless literal now retains the length unit.

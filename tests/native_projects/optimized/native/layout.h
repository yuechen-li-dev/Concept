#ifndef CONCEPT_OPTIMIZED_LAYOUT_H
#define CONCEPT_OPTIMIZED_LAYOUT_H
#ifndef __OPTIMIZE__
#error ABI evidence must use the target optimization mode.
#endif
typedef struct OptimizedLayout { int value; } OptimizedLayout;
#endif

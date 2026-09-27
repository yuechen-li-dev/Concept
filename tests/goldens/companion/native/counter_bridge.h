#ifndef GOLDEN_COUNTER_BRIDGE_H
#define GOLDEN_COUNTER_BRIDGE_H

#ifdef __cplusplus
extern "C" {
#endif

typedef struct CounterStats {
    int value;
    int resets;
} CounterStats;

void *ConceptCounterCreate(void);
void ConceptCounterDestroy(void *counter);
void ConceptCounterAdd(void *counter, int delta);
void ConceptCounterReset(void *counter);
CounterStats ConceptCounterSnapshot(void *counter);

#ifdef __cplusplus
}
#endif

#endif

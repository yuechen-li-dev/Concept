#include "counter_bridge.h"
#include <cstdlib>
#include <new>

// The C++ component remains native; Concept sees only this stable C seam.
class Counter {
public:
    void Add(int delta) { value_ += delta; }
    void Reset() { value_ = 0; ++resets_; }
    CounterStats Snapshot() const { return {value_, resets_}; }

private:
    int value_ = 0;
    int resets_ = 0;
};

extern "C" void *ConceptCounterCreate(void) {
    Counter *counter = new (std::nothrow) Counter();
    if (counter == nullptr) { std::abort(); }
    return counter;
}
extern "C" void ConceptCounterDestroy(void *counter) { delete static_cast<Counter *>(counter); }
extern "C" void ConceptCounterAdd(void *counter, int delta) { static_cast<Counter *>(counter)->Add(delta); }
extern "C" void ConceptCounterReset(void *counter) { static_cast<Counter *>(counter)->Reset(); }
extern "C" CounterStats ConceptCounterSnapshot(void *counter) { return static_cast<Counter *>(counter)->Snapshot(); }

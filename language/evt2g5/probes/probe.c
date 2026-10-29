/* Independently compiled Microsoft x64 C callers and callees. No Cathedral
 * headers, classifiers, frame descriptions or generated source are included. */
#include <stdint.h>
#include <string.h>
#define EXPORT __declspec(dllexport)
typedef struct One { uint8_t value; } One;
typedef struct Two { uint16_t value; } Two;
typedef struct Four { int32_t value; } Four;
typedef struct Eight { double value; } Eight;
typedef struct Large { double a, b, c; } Large;
typedef struct MutableLarge { double a, b, c; } MutableLarge;
typedef struct ArrayThree { int32_t values[3]; } ArrayThree;
_Static_assert(sizeof(Four) == 4, "direct four");
_Static_assert(sizeof(Eight) == 8, "floating member uses integer aggregate carrier");
_Static_assert(sizeof(Large) == 24, "indirect record");
EXPORT uint64_t observed[16];
void *forwardTarget;
__declspec(align(16)) uint64_t savedVector[2] = {0x0123456789abcdefULL, 0xfedcba9876543210ULL};
double seedDouble = 3.0;
EXPORT One ProbeOne(One value) { return value; }
EXPORT Two ProbeTwo(Two value) { return value; }
EXPORT Eight ProbeEight(Eight value) { return value; }
/* The assembly entry records actual RSP, physical register/stack arguments and
 * addresses before the C compiler can materialize its own parameter copies. */
EXPORT Large ProbeMixedEntry(Four small, Large large, double delta, int *count, float bias, const Four *view);
EXPORT Large ProbeRoundForward(Large value);
EXPORT int ProbeSaved(void *target);
Large ProbeMixedBody(Four small, Large large, double delta, int *count, float bias, const Four *view) {
    observed[10] = (uint64_t)(uintptr_t)&large;
    *count += small.value + view->value;
    large.a += delta;
    large.b += (double)bias;
    large.c += (double)small.value;
    return large;
}
EXPORT Large ProbeShifted(int a, double b, int c, float d) {
    observed[11] = (uint64_t)(uint32_t)a;
    observed[12] = (uint64_t)(uint32_t)c;
    return (Large){b, (double)a, (double)c + (double)d};
}
/* Physical ABI probe: the source value parameter is indirect in RCX; explicit
 * reference is RDX. This observes the actual passed temporary, not a C copy. */
EXPORT int ProbeIsolation(MutableLarge *temporary, MutableLarge *original) {
    observed[13] = (uint64_t)(uintptr_t)temporary;
    observed[14] = (uint64_t)(uintptr_t)original;
    if (temporary == original || ((uintptr_t)temporary & 15u) != 0) return 0;
    temporary->a = 91.0;
    original->b += 2.0;
    return 1;
}
EXPORT uint64_t Observation(int index) {
    return index >= 0 && index < 16 ? observed[index] : UINT64_MAX;
}
EXPORT void ForwardTo(void *entry) {
    forwardTarget = entry;
}
/* Every call below is emitted by this independent C compiler. An assembly
 * tail-forwarder lets the test observe its true indirect pointer alignment
 * while the unmodified Cathedral callee still executes. */
EXPORT int RunCallers(void **entries, int seed) {
    typedef One (*OneFn)(One);
    typedef Two (*TwoFn)(Two);
    typedef Four (*FourFn)(Four);
    typedef Eight (*EightFn)(Eight);
    typedef Large (*LargeFn)(Large);
    typedef Large (*ShiftFn)(int, double, int, float);
    typedef Large (*MixedFn)(Four, Large, double, int *, float, const Four *);
    typedef Four (*EightArgs)(Four, Four, Four, Four, Four, Four, Four, Four);
    typedef Four (*SmallRec)(int, Four);
    typedef Large (*LargeRec)(int, Large);
    typedef int (*AtFn)(ArrayThree, int);
    OneFn one; TwoFn two; FourFn four; EightFn eight; LargeFn large;
    ShiftFn shifted; MixedFn mixed; EightArgs args; SmallRec smallRec; LargeRec largeRec; AtFn at;
    memcpy(&one, &entries[0], sizeof one); memcpy(&two, &entries[1], sizeof two);
    memcpy(&four, &entries[2], sizeof four); memcpy(&eight, &entries[3], sizeof eight);
    memcpy(&large, &entries[4], sizeof large); memcpy(&shifted, &entries[5], sizeof shifted);
    memcpy(&mixed, &entries[6], sizeof mixed); memcpy(&args, &entries[7], sizeof args);
    memcpy(&smallRec, &entries[8], sizeof smallRec); memcpy(&largeRec, &entries[9], sizeof largeRec);
    memcpy(&at, &entries[10], sizeof at);
    if (one((One){(uint8_t)seed}).value != (uint8_t)seed) return 1;
    if (two((Two){(uint16_t)(seed * 257)}).value != (uint16_t)(seed * 257)) return 2;
    if (four((Four){seed}).value != seed) return 3;
    double x = (double)seed / 8.0;
    Eight e = eight((Eight){x});
    if (memcmp(&e.value, &x, sizeof x)) return 4;
    Large input = {x, x + 1.0, x + 2.0};
    Large result = large(input);
    if (result.a != input.a || result.b != input.b || result.c != input.c) return 5;
    result = shifted(2, x, 1, 3.0f);
    if (result.a != x || result.b != 1.0 || result.c != 2.0) return 6;
    int count = seed;
    Four view = {4};
    result = mixed((Four){3}, input, 0.5, &count, 1.0f, &view);
    if (count != seed + 7 || result.a != x + 0.5 || result.b != x + 1.0 || result.c != x + 3.0) return 7;
    Four f = {seed};
    if (args(f,f,f,f,f,f,f,f).value != seed * 8) return 8;
    if (smallRec(3,f).value != seed + 3) return 9;
    result = largeRec(3,input);
    if (result.a != input.a + 3.0 || result.b != input.b || result.c != input.c) return 10;
    if (at((ArrayThree){{seed,seed+1,seed+2}},2) != seed + 2) return 11;
    return 0;
}
EXPORT int RunRawSret(void *entry, int seed) {
    typedef Large *(*Raw)(Large *, int, double, int, float);
    Raw call;
    memcpy(&call, &entry, sizeof call);
    struct Guarded { uint64_t before; Large value; uint64_t after; } storage = {0x1122334455667788ULL,{0,0,0},0x8877665544332211ULL};
    Large *returned = call(&storage.value,seed,(double)seed/8.0,seed-1,2.0f);
    return returned != &storage.value || storage.value.a != (double)seed/8.0 ||
        storage.value.b != 1.0 || storage.value.c != 2.0 ||
        storage.before != 0x1122334455667788ULL || storage.after != 0x8877665544332211ULL;
}

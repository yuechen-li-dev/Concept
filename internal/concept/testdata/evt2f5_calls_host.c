#include "amd64.generated.h"
#include <windows.h>
#include <stdint.h>
#include <stdio.h>
#include <string.h>
#include <time.h>
#include <xmmintrin.h>
#define F(name) concept_standard__backend__amd64_##name
#define REQUIRE(condition) do { if (!(condition)) { fprintf(stderr, "mixed call check failed line %d: %s\n", __LINE__, #condition); return 1; } } while (0)
/* ARTIFACT */
static unsigned char code[65536], again[65536];
static concept_native_image image, other;
static void *executable;
#define ENTRY(type, name, ordinal) type name = NULL; void *name##_address = (unsigned char *)executable + image.symbols.data[ordinal].offset; memcpy(&name, &name##_address, sizeof name)
typedef float (*Float1)(float);
typedef double (*Double1)(double);
typedef float (*Float2)(float, float);
typedef double (*Double2)(double, double);
typedef double (*MixedA)(int, double, int, float);
typedef double (*MixedB)(double, int, float, int);
typedef double (*Mixed8)(int, double, _Bool, float, double, float, int, _Bool);
typedef double (*Double3)(double, double, double);
typedef float (*Float5)(float, float, float, float, float);
typedef float (*Float8)(float, float, float, float, float, float, float, float);
typedef double (*Double8)(double, double, double, double, double, double, double, double);
typedef double (*MixedLive)(int, float, double);
typedef double (*MixedResults)(int, double);
typedef _Bool (*Bool1)(_Bool);
typedef _Bool (*CompareCalls)(float, float);
typedef void (*VoidMixed)(int, float, double);
typedef double (*Double0)(void);
typedef float (*Float0)(void);
typedef double (*PressureMixed)(int, float, double, float, double, float, double, _Bool);
typedef double (*Double6)(double, double, double, double, double, double);
typedef float (*Float7)(float, float, float, float, float, float, float);
typedef float (*EntryPressure)(float, float, float, int, float, float, float, float);
/* INVALID_CALL_CHECKS */

static uint32_t bits32(float x) { uint32_t bits; memcpy(&bits, &x, sizeof bits); return bits; }
static uint64_t bits64(double x) { uint64_t bits; memcpy(&bits, &x, sizeof bits); return bits; }
static float float32(uint32_t bits) { float x; memcpy(&x, &bits, sizeof x); return x; }
static double float64(uint64_t bits) { double x; memcpy(&x, &bits, sizeof x); return x; }

// Independently compiled, strict C11 expressions; no generated ABI planner
// supplies their parameter placement or arithmetic results.
static double oracle_nested_g(double a, double b) { double local = a - b; double x = b + 1.0000000000000002; return local + x; }
static double oracle_nested_f(double a, double b) { double local = a + b; double x = oracle_nested_g(b, a); return local + x; }
static double oracle_multiple(double a, double b) { double local = a - b; double x = b; double y = oracle_nested_g(a, x); return local + x + y; }
static double oracle_a(int a, double b, int c, float d) { return a == 17 && c == 29 && d == 3.5f ? b : -b; }
static double oracle_b(double a, int b, float c, int d) { return b == 17 && c == 3.5f && d == 29 ? a : -a; }
static double oracle_eight(int a, double b, _Bool c, float d, double e, float f, int g, _Bool h)
{
    return a == 17 && c && d == 3.5f && f == 1.5f && g == 29 && h ? b - e : b + e;
}

static volatile uint64_t probe_rsp;
static volatile int probe_seen, void_seen;
__attribute__((used, noinline)) static double abi_probe_c(int a, double b, _Bool c, float d, double e, float f, int g, _Bool h)
{
    probe_seen = a == 17 && bits64(b) == bits64(2.5) && c && bits32(d) == bits32(3.5f)
        && bits64(e) == bits64(7.25) && bits32(f) == bits32(1.5f) && g == 29 && h;
    return b - e;
}
// Capture unmodified entry RSP and exercise all 32 caller-provided home bytes.
// The typed C helper then independently reads the registers and stack slots.
__attribute__((naked)) static double abi_probe(int a, double b, _Bool c, float d, double e, float f, int g, _Bool h)
{
    __asm__(".intel_syntax noprefix\n"
        "mov rax,rsp\n mov QWORD PTR probe_rsp[rip],rax\n"
        "mov QWORD PTR [rsp+8],1\n mov QWORD PTR [rsp+16],2\n"
        "mov QWORD PTR [rsp+24],3\n mov QWORD PTR [rsp+32],4\n"
        "jmp abi_probe_c\n .att_syntax prefix\n");
}
static void void_probe(int i, float f, double d)
{
    void_seen = i == 17 && bits32(f) == bits32(3.5f) && bits64(d) == UINT64_C(0x3ff0000000000001);
}
__attribute__((naked)) static float clobber32(float x)
{
    __asm__(".intel_syntax noprefix\n movss xmm1,xmm0\n movss xmm2,xmm0\n movss xmm3,xmm0\n movss xmm4,xmm0\n movss xmm5,xmm0\n ret\n .att_syntax prefix\n");
}
__attribute__((naked)) static double clobber64(double x)
{
    __asm__(".intel_syntax noprefix\n movsd xmm1,xmm0\n movsd xmm2,xmm0\n movsd xmm3,xmm0\n movsd xmm4,xmm0\n movsd xmm5,xmm0\n ret\n .att_syntax prefix\n");
}
__attribute__((naked)) static int guarded(void *target)
{
    __asm__(".intel_syntax noprefix\n"
        "push rbx\n push rbp\n push rsi\n push rdi\n push r12\n push r13\n push r14\n push r15\n sub rsp,72\n"
        "mov r10,rcx\n mov QWORD PTR [rsp+40],rsp\n mov QWORD PTR [rsp+48],0x7654321\n"
        "mov ebx,0x1111\n mov ebp,0x2222\n mov esi,0x3333\n mov edi,0x4444\n"
        "mov r12d,0x5555\n mov r13d,0x6666\n mov r14d,0x7777\n mov r15d,0x8888\n"
        "call r10\n cmp rsp,QWORD PTR [rsp+40]\n jne 1f\n cmp QWORD PTR [rsp+48],0x7654321\n jne 1f\n"
        "cmp rbx,0x1111\n jne 1f\n cmp rbp,0x2222\n jne 1f\n cmp rsi,0x3333\n jne 1f\n cmp rdi,0x4444\n jne 1f\n"
        "cmp r12,0x5555\n jne 1f\n cmp r13,0x6666\n jne 1f\n cmp r14,0x7777\n jne 1f\n cmp r15,0x8888\n jne 1f\n"
        "mov eax,1\n jmp 2f\n 1: xor eax,eax\n 2: add rsp,72\n pop r15\n pop r14\n pop r13\n pop r12\n pop rdi\n pop rsi\n pop rbp\n pop rbx\n ret\n .att_syntax prefix\n");
}

// Controlled harness-only foreign target substitution. Production images never
// contain a host address. These test stubs are installed while RW, before RX.
static int foreign_stub(int ordinal, const void *pointer_bytes, size_t size)
{
    if (size != sizeof(uint64_t) || image.symbols.data[ordinal].size < 12) return 0;
    unsigned char *p = code + image.symbols.data[ordinal].offset;
    p[0] = 0x48; p[1] = 0xb8;
    memcpy(p + 2, pointer_bytes, size);
    p[10] = 0xff; p[11] = 0xe0;
    return 1;
}

static int execute_cases(void)
{
    ENTRY(MixedA, a, ORD_CallA);
    ENTRY(MixedB, b, ORD_CallB);
    ENTRY(Mixed8, eight, ORD_Call8);
    ENTRY(Float2, live32, ORD_Live32);
    ENTRY(Double2, live64, ORD_Live64);
    ENTRY(Float2, cycle2, ORD_Cycle2);
    ENTRY(Double3, cycle3, ORD_Cycle3);
    ENTRY(Float2, repeat, ORD_Repeat);
    ENTRY(Float5, five, ORD_CallFive);
    ENTRY(Float8, pressure32, ORD_CallPressure32);
    ENTRY(Double8, pressure64, ORD_CallPressure64);
    ENTRY(MixedLive, mixed_live, ORD_MixedLive);
    ENTRY(MixedResults, mixed_results, ORD_MixedResults);
    ENTRY(Double2, nested, ORD_NestedF);
    ENTRY(Double2, multiple, ORD_Multiple);
    ENTRY(Bool1, boolean, ORD_CallReturnBool);
    ENTRY(CompareCalls, compare, ORD_CompareCalls);
    ENTRY(VoidMixed, nothing, ORD_CallVoid);
    ENTRY(Float0, zero32, ORD_CallZero32);
    ENTRY(Double0, zero64, ORD_CallZero64);
    ENTRY(MixedResults, recursive, ORD_Recursive);
    ENTRY(PressureMixed, mixed_pressure, ORD_MixedPressure);
    ENTRY(Double6, six, ORD_CallSix);
    ENTRY(Float7, seven, ORD_CallSeven);
    ENTRY(EntryPressure, entry_pressure, ORD_CallEntryPressure);
    ENTRY(Double0, gpr, ORD_GPRDriver);
    REQUIRE(bits64(a(17, 1.0000000000000002, 29, 3.5f)) == UINT64_C(0x3ff0000000000001));
    REQUIRE(a(18, 2.5, 29, 3.5f) == -2.5);
    REQUIRE(a(17, 2.5, 30, 3.5f) == -2.5);
    REQUIRE(a(17, 2.5, 29, 4.5f) == -2.5);
    REQUIRE(bits64(b(1.0000000000000002, 17, 3.5f, 29)) == UINT64_C(0x3ff0000000000001));
    REQUIRE(b(2.5, 18, 3.5f, 29) == -2.5);
    REQUIRE(b(2.5, 17, 4.5f, 29) == -2.5);
    REQUIRE(b(2.5, 17, 3.5f, 30) == -2.5);
    REQUIRE(eight(17, 2.5, 1, 3.5f, 7.25, 1.5f, 29, 1) == -4.75);
    REQUIRE(eight(17, 2.5, 1, 3.5f, 7.25, 1.5f, 29, 0) == 9.75);
    REQUIRE(eight(17, 2.5, 1, 3.5f, 7.25, 2.5f, 29, 1) == 9.75);
    REQUIRE(eight(17, 2.5, 1, 3.5f, 7.25, 1.5f, 30, 1) == 9.75);
    REQUIRE(eight(17, 2.5, 0, 3.5f, 7.25, 1.5f, 29, 1) == 9.75);
    REQUIRE(live32(7.5f, 2.0f) == 17.0f);
    REQUIRE(bits64(live64(1.0000000000000002, 2.0)) == bits64((1.0000000000000002 + 2.0) + 1.0000000000000002));
    REQUIRE(cycle2(2.0f, 7.5f) == 5.5f);
    REQUIRE(cycle3(2.0, 7.5, 3.0) == (7.5 - 3.0) / 2.0);
    REQUIRE(repeat(2.0f, 7.5f) == 5.5f);
    REQUIRE(five(2, 7, 3, 11, 5) == -8);
    REQUIRE(pressure32(1, 2, 3, 4, 5, 6, 7, 8) == 37);
    REQUIRE(pressure64(1, 2, 3, 4, 5, 6, 7, 8) == 37);
    REQUIRE(bits64(mixed_live(17, 3.5f, 2.5)) == bits64((2.5 + 1.0000000000000002) - 2.5));
    REQUIRE(mixed_results(17, 2.5) == 5.0 && mixed_results(18, 2.5) == 0.0);
    REQUIRE(bits64(nested(7.5, 2.0)) == bits64(oracle_nested_f(7.5, 2.0)));
    REQUIRE(bits64(multiple(7.5, 2.0)) == bits64(oracle_multiple(7.5, 2.0)));
    REQUIRE(boolean(0) == 0 && boolean(1) == 1);
    REQUIRE(compare(2, 3) == 1 && compare(3, 2) == 0);
    nothing(17, 3.5f, 1.0000000000000002);
    REQUIRE(bits32(zero32()) == UINT32_C(0x3dcccccd));
    REQUIRE(bits64(zero64()) == UINT64_C(0x3ff0000000000001));
    REQUIRE(recursive(4, 2.5) == 12.5);
    REQUIRE(six(1, 3, 5, 7, 11, 13) == -6);
    REQUIRE(seven(1, 3, 5, 7, 11, 13, 17) == 11);
    REQUIRE(entry_pressure(1, 2, 3, 29, 4, 5, 6, 7) == 28);
    REQUIRE(entry_pressure(1, 2, 3, 30, 4, 5, 6, 7) == -1);
    REQUIRE(bits64(gpr()) == UINT64_C(0x3ff0000000000001));
    REQUIRE(mixed_pressure(17, 2, 3, 4, 5, 6, 7, 1) == 23);
    REQUIRE(mixed_pressure(17, 2, 3, 4, 5, 6, 7, 0) == 7);
    for (int i = 0; i < 8; i++) {
        int first = i & 1 ? 18 : 17, third = i & 2 ? 30 : 29;
        float last = i & 4 ? 4.5f : 3.5f;
        double precise = 1.0000000000000002;
        REQUIRE(bits64(a(first, precise, third, last)) == bits64(oracle_a(first, precise, third, last)));
        REQUIRE(bits64(b(precise, first, last, third)) == bits64(oracle_b(precise, first, last, third)));
        REQUIRE(bits64(eight(first, precise, 1, last, 7.25, 1.5f, third, 1)) == bits64(oracle_eight(first, precise, 1, last, 7.25, 1.5f, third, 1)));
    }
    ENTRY(Float1, pass32, ORD_CallReturn32);
    ENTRY(Double1, pass64, ORD_CallReturn64);
    const uint32_t values32[] = {0, 0x80000000, 0x7f800000, 0xff800000, 0x7fc01234, 0x3f800001};
    const uint64_t values64[] = {0, UINT64_C(0x8000000000000000), UINT64_C(0x7ff0000000000000), UINT64_C(0xfff0000000000000), UINT64_C(0x7ff8000000001234), UINT64_C(0x3ff0000000000001)};
    for (int i = 0; i < 6; i++) {
        REQUIRE(bits32(pass32(float32(values32[i]))) == values32[i]);
        REQUIRE(bits64(pass64(float64(values64[i]))) == values64[i]);
    }
    return 0;
}

static uint64_t mix(uint64_t hash, int value)
{
    return (hash ^ (uint32_t)value) * UINT64_C(1099511628211);
}
static uint64_t location_hash(uint64_t hash, concept_scalar_location a)
{
    hash = mix(hash, a.kind.tag); hash = mix(hash, a.registerClass.tag);
    hash = mix(hash, a.gpr.tag); hash = mix(hash, a.xmm.tag);
    return mix(hash, a.slot);
}
static int plan_signature(int ordinal, uint64_t *signature, int print)
{
    concept_readonly_span_byte input = {artifact, sizeof artifact};
    concept_backend_function function = F(empty_function)();
    REQUIRE(!F(decode_scalar_function)(input, ordinal, &function).tag);
    concept_allocation allocation = F(empty_allocation)();
    REQUIRE(!F(allocate_registers)(&function, &allocation).tag);
    REQUIRE(!F(plan_function_calls)(&function, &allocation).tag);
    concept_finalized_frame frame = F(empty_finalized_frame)();
    REQUIRE(!F(qualify_scalar_native_frame)(&function, &allocation, &frame).tag);
    concept_liveness live = F(empty_liveness)();
    REQUIRE(!F(compute_liveness)(&function, &live).tag);
    uint64_t hash = UINT64_C(14695981039346656037);
    for (int i = 0; i < function.virtualRegCount; i++) {
        hash = location_hash(hash, allocation.scalarLocations.data[i]);
        hash = mix(hash, allocation.start.data[i]); hash = mix(hash, allocation.end.data[i]);
        hash = mix(hash, allocation.assigned.data[i]); hash = mix(hash, allocation.spilled.data[i]);
    }
    hash = mix(hash, frame.byteCount); hash = mix(hash, frame.nativeTemporaryBase);
    for (int i = 0; i < frame.slotCount; i++) {
        concept_frame_slot slot = frame.slots.data[i];
        hash = mix(hash, slot.category.tag); hash = mix(hash, slot.identity);
        hash = mix(hash, slot.offset); hash = mix(hash, slot.size); hash = mix(hash, slot.alignment);
    }
    int max_actions = 0, spills = 0, cycle_temporaries = 0;
    for (int i = 0; i < function.virtualRegCount; i++) spills += frame.spillSlots.data[i] >= 0;
    if (ordinal == ORD_GPRPressure) {
        int saved = 0;
        for (int i = 0; i < 15; i++) saved += frame.saveSlots.data[i] >= 0;
        REQUIRE(saved >= 3);
    }
    for (int p = 0; p < function.instructionCount; p++) {
        if (function.instructions.data[p].op.tag != 16) continue;
        concept_call_site_actions site = F(empty_call_site_actions)();
        REQUIRE(!F(realize_allocated_call)(&function, &allocation, &live, p, &frame, &site).tag);
        if (site.count > max_actions) max_actions = site.count;
        cycle_temporaries += site.abi.temporaryCount;
        hash = mix(hash, site.count); hash = mix(hash, site.callIndex);
        for (int j = 0; j < site.count; j++) {
            hash = mix(hash, site.actions.data[j].kind.tag);
            hash = mix(hash, site.actions.data[j].width);
            hash = mix(hash, site.scalarActions.data[j].representation.tag);
            hash = location_hash(hash, site.scalarActions.data[j].source);
            hash = location_hash(hash, site.scalarActions.data[j].destination);
            hash = mix(hash, site.sourceOffsets.data[j]); hash = mix(hash, site.destinationOffsets.data[j]);
        }
        if (ordinal == ORD_Cycle3) REQUIRE(site.abi.temporaryCount == 1);
        if (ordinal == ORD_Cycle2) REQUIRE(site.abi.temporaryCount == 1);
        if (ordinal == ORD_Pressure64 || ordinal == ORD_Pressure32) {
            REQUIRE(spills >= 8 && site.callIndex >= 6);
            // At least six old physical XMM values survive a hostile callee.
            REQUIRE(site.count >= 14);
        }
        if (print && ordinal == ORD_Call8) {
            concept_call_site_actions bad = site;
            bad.destinationOffsets.data[0]++;
            concept_byte_writer writer = {{again, sizeof again}, 0};
            REQUIRE(F(emit_scalar_call_action)(&writer, &bad, 0, &frame).tag && writer.offset == 0);
            bad = site; bad.actions.data[0].width = 3;
            REQUIRE(F(emit_scalar_call_action)(&writer, &bad, 0, &frame).tag && writer.offset == 0);
            bad = site; bad.scalarActions.data[0].source.registerClass.tag = 0;
            REQUIRE(F(emit_scalar_call_action)(&writer, &bad, 0, &frame).tag && writer.offset == 0);
            bad = site; bad.actions.data[0].kind.tag = 7;
            REQUIRE(F(emit_scalar_call_action)(&writer, &bad, 0, &frame).tag && writer.offset == 0);
            bad = site; bad.count = 65;
            REQUIRE(F(emit_scalar_call_action)(&writer, &bad, 64, &frame).tag && writer.offset == 0);
            concept_finalized_frame broken = frame;
            broken.slots.data[broken.outgoingSlots.data[0]].size = 4;
            REQUIRE(F(emit_scalar_call_action)(&writer, &site, 0, &broken).tag && writer.offset == 0);
            broken = frame;
            broken.nativeTemporaryBase++;
            REQUIRE(F(validate_function_frame)(&function, &allocation, &live, &broken).tag);
            bad = F(empty_call_site_actions)(); bad.count = 63;
            REQUIRE(!F(append_call_action)(&bad, F(empty_frame_action)()).tag && bad.count == 64);
            REQUIRE(F(append_call_action)(&bad, F(empty_frame_action)()).tag && bad.count == 64);
            REQUIRE(F(resolve_frame_slot)(&frame, (concept_frame_region_kind){5}, 8).tag);
        }
    }
    if (print) printf("plan ordinal=%d frame=%d spills=%d scratch-base=%d cycles=%d actions=%d hash=%016llx\n",
        ordinal, frame.byteCount, spills, frame.nativeTemporaryBase, cycle_temporaries, max_actions, (unsigned long long)hash);
    *signature = hash;
    return 0;
}

int main(void)
{
    unsigned int control = _mm_getcsr();
    REQUIRE((control & 0x6000) == 0 && (control & 0x1f80) == 0x1f80 && (control & 0x8040) == 0);
    REQUIRE(!reject_bad_calls());
    concept_readonly_span_byte input = {artifact, sizeof artifact};
    concept_span_byte output = {code, sizeof code};
    clock_t started = clock();
    concept_result_int_native_error result = F(emit_module)(input, output, &image);
    if (result.tag) fprintf(stderr, "emission: %s\n", F(describe_native_error)(result.payload.error.error));
    REQUIRE(!result.tag && image.finalized && image.fixupCount > 0);
    double emission_ms = 1000.0 * (clock() - started) / CLOCKS_PER_SEC;
    started = clock();
    for (int ordinal = 0; ordinal < image.symbolCount; ordinal++) {
        uint64_t baseline;
        REQUIRE(!plan_signature(ordinal, &baseline, 1));
        if (ordinal == ORD_Call8 || ordinal == ORD_Cycle3 || ordinal == ORD_MixedPressure || ordinal == ORD_Pressure64) {
            for (int run = 0; run < 100; run++) {
                uint64_t current;
                REQUIRE(!plan_signature(ordinal, &current, 0) && current == baseline);
            }
        }
    }
    printf("timing mixed planning/realization/determinism %.3f ms; module emission %.3f ms\n",
        1000.0 * (clock() - started) / CLOCKS_PER_SEC, emission_ms);
    memcpy(again, code, (size_t)image.byteCount);
    for (int run = 0; run < 100; run++) {
        concept_span_byte repeated = {code, sizeof code};
        result = F(emit_module)(input, repeated, &other);
        REQUIRE(!result.tag && !memcmp(&image, &other, sizeof image));
        REQUIRE(!memcmp(code, again, (size_t)image.byteCount));
    }
    static unsigned char fixup_code[65536];
    concept_bridge_reader reader = {input, 0};
    started = clock();
    for (int run = 0; run < 100; run++) {
        other = image;
        other.finalized = 0;
        memcpy(fixup_code, code, (size_t)image.byteCount);
        for (int i = 0; i < other.fixupCount; i++) {
            other.fixups.data[i].resolved = 0;
            memset(fixup_code + other.fixups.data[i].patchOffset, 0, 4);
        }
        concept_span_byte pending = {fixup_code, sizeof fixup_code};
        REQUIRE(!F(resolve_native_calls)(&reader, &other, pending).tag);
        REQUIRE(!memcmp(fixup_code, code, (size_t)image.byteCount));
    }
    printf("timing 100 fixup resolutions %.3f ms\n", 1000.0 * (clock() - started) / CLOCKS_PER_SEC);
    printf("image bytes=%d symbols=%d calls=%d hex=", image.byteCount, image.symbolCount, image.fixupCount);
    for (int i = 0; i < image.byteCount; i++) printf("%02x", code[i]);
    puts("");
    executable = VirtualAlloc(NULL, (size_t)image.byteCount, MEM_RESERVE | MEM_COMMIT, PAGE_READWRITE);
    REQUIRE(executable);
    memcpy(executable, code, (size_t)image.byteCount);
    DWORD previous;
    REQUIRE(VirtualProtect(executable, (size_t)image.byteCount, PAGE_EXECUTE_READ, &previous));
    REQUIRE(FlushInstructionCache(GetCurrentProcess(), executable, (size_t)image.byteCount));
    REQUIRE(!execute_cases());
    void *driver = (unsigned char *)executable + image.symbols.data[ORD_ProbeDriver].offset;
    REQUIRE(guarded(driver));
    driver = (unsigned char *)executable + image.symbols.data[ORD_GPRDriver].offset;
    REQUIRE(guarded(driver));
    ENTRY(Double2, timed, ORD_Multiple);
    started = clock();
    volatile double total = 0;
    for (int run = 0; run < 100000; run++) total += timed(7.5, 2.0);
    REQUIRE(total == 100000 * oracle_multiple(7.5, 2.0));
    printf("timing 100000 nested native calls %.3f ms\n", 1000.0 * (clock() - started) / CLOCKS_PER_SEC);
    REQUIRE(VirtualFree(executable, 0, MEM_RELEASE));

    Mixed8 probe = abi_probe;
    VoidMixed void_target = void_probe;
    Float1 hostile32 = clobber32;
    Double1 hostile64 = clobber64;
    REQUIRE(foreign_stub(ORD_Mixed8, &probe, sizeof probe));
    REQUIRE(foreign_stub(ORD_VoidMixed, &void_target, sizeof void_target));
    REQUIRE(foreign_stub(ORD_Return32, &hostile32, sizeof hostile32));
    REQUIRE(foreign_stub(ORD_Return64, &hostile64, sizeof hostile64));
    executable = VirtualAlloc(NULL, (size_t)image.byteCount, MEM_RESERVE | MEM_COMMIT, PAGE_READWRITE);
    REQUIRE(executable);
    memcpy(executable, code, (size_t)image.byteCount);
    REQUIRE(VirtualProtect(executable, (size_t)image.byteCount, PAGE_EXECUTE_READ, &previous));
    REQUIRE(FlushInstructionCache(GetCurrentProcess(), executable, (size_t)image.byteCount));
    // The external probe has one fixed accepted vector. Run the preservation
    // and side-effect subset against these independently compiled targets.
    ENTRY(Double0, external_driver, ORD_ProbeDriver);
    ENTRY(VoidMixed, external_void, ORD_CallVoid);
    ENTRY(Float8, external_pressure32, ORD_CallPressure32);
    ENTRY(Double8, external_pressure64, ORD_CallPressure64);
    REQUIRE(external_driver() == -4.75 && probe_seen && (probe_rsp & 15) == 8);
    external_void(17, 3.5f, 1.0000000000000002);
    REQUIRE(void_seen);
    REQUIRE(external_pressure32(1, 2, 3, 4, 5, 6, 7, 8) == 37);
    REQUIRE(external_pressure64(1, 2, 3, 4, 5, 6, 7, 8) == 37);
    REQUIRE(VirtualFree(executable, 0, MEM_RELEASE));
    puts("independent Win64 mixed-eight ABI/entry alignment/shadow/void side effect/volatile XMM probe passed");
    puts("native mixed calls passed");
    return 0;
}

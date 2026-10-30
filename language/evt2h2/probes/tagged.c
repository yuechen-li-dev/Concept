/* Independent Microsoft x64 C callers/callees. No Concept generated headers. */
#include <stdint.h>
#include <string.h>
#define EXPORT __declspec(dllexport)
typedef struct { int count; double amount; } Item;
typedef struct { int count; _Bool ready; Item item; int numbers[3]; } Many;
typedef union { int integer; double decimal; float real; _Bool flag; Many many; int stamp; } Payload;
typedef struct { uint32_t tag; Payload payload; } Token;
typedef struct { uint32_t tag; int value; } Small;
typedef struct { uint32_t tag; int first, second; } Ordered;
_Static_assert(sizeof(Token) == 48, "tagged indirect shape");
_Static_assert(sizeof(Small) == 8, "tagged direct shape");
_Static_assert(sizeof(Ordered) == 12, "tagged hidden return shape");
typedef Token (*Make)(int);
typedef Small (*MakeSmall)(int);
typedef Ordered (*MakeOrdered)(int *);
typedef Token (*Mixed)(Token, int, double, Token, float, _Bool, Token);
static int calls;
EXPORT Token TaggedRelay(Token value) { calls++; return value; }
EXPORT Small SmallRelay(Small value) { calls++; return value; }
EXPORT int TaggedCalls(void) { return calls; }
EXPORT int CallTagged(const void *const *table) {
    Make make, nested;
    MakeSmall other, twin;
    MakeOrdered ordered;
    Mixed mixed;
    memcpy(&make, table + 0, sizeof make);
    memcpy(&other, table + 1, sizeof other);
    memcpy(&twin, table + 2, sizeof twin);
    memcpy(&ordered, table + 3, sizeof ordered);
    memcpy(&mixed, table + 4, sizeof mixed);
    memcpy(&nested, table + 5, sizeof nested);
    for (int i = 0; i < 100; i++) {
        Token a = make(i), b = make(-i), c = nested(i);
        Small x = other(i), y = twin(-i);
        int counter = i;
        Ordered z = ordered(&counter);
        if (a.tag != 1 || a.payload.integer != i || b.payload.integer != -i ||
            x.tag != 1 || x.value != i || y.tag != 1 || y.value != -i ||
            z.tag != 0 || z.first != i + 1 || z.second != i + 2 || counter != i + 2)
            return 1;
        Token v = mixed(c, 3, 2.5, b, 1.25f, 1, a);
        if (v.tag != 5 || v.payload.many.count != i || v.payload.many.item.count != i + 1 ||
            v.payload.many.item.amount != 3.5 || v.payload.many.numbers[2] != 6)
            return 2;
        v = mixed(c, 3, 2.5, b, 1.25f, 0, a);
        if (v.tag != 1 || v.payload.integer != 2) return 3;
    }
    calls = 0;
    return 0;
}

/* Independent Microsoft x64 caller/callees; no generated Concept headers. */
#include <stdint.h>
#include <string.h>
#define EXPORT __declspec(dllexport)
typedef struct { int count; double amount; } Item;
typedef struct { int tag, value; } OInt;
typedef struct { int tag; double value; } ODouble;
typedef struct { int tag; float value; } OFloat;
typedef struct { int tag; _Bool value; } OBool;
typedef struct { int tag; Item value; } OItem;
typedef struct { int tag; int value[3]; } OArray;
_Static_assert(sizeof(OInt)==8 && sizeof(OBool)==8 && sizeof(OFloat)==8,"direct shapes");
_Static_assert(sizeof(ODouble)==16 && sizeof(OItem)==24 && sizeof(OArray)==16,"indirect shapes");
typedef OInt (*MakeInt)(_Bool,int);
typedef ODouble (*MakeDouble)(_Bool,double);
typedef OFloat (*MakeFloat)(_Bool,float);
typedef OBool (*MakeBool)(_Bool,_Bool);
typedef OItem (*MakeItem)(_Bool,int);
typedef OArray (*MakeArray)(_Bool);
typedef ODouble (*Mixed)(OInt,int,double,int*,ODouble,float,_Bool);
EXPORT OInt OptionIntRelay(OInt value) { return value; }
EXPORT ODouble OptionDoubleRelay(ODouble value) { return value; }
EXPORT OFloat OptionFloatRelay(OFloat value) { return value; }
EXPORT OBool OptionBoolRelay(OBool value) { return value; }
EXPORT OItem OptionItemRelay(OItem value) { return value; }
EXPORT OArray OptionArrayRelay(OArray value) { return value; }
EXPORT int CallOptions(const void *const *table) {
    MakeInt makeInt; MakeDouble makeDouble; MakeFloat makeFloat;
    MakeBool makeBool; MakeItem makeItem; MakeArray makeArray; Mixed mixed;
    memcpy(&makeInt,table+0,sizeof makeInt); memcpy(&makeDouble,table+1,sizeof makeDouble);
    memcpy(&makeFloat,table+2,sizeof makeFloat); memcpy(&makeBool,table+3,sizeof makeBool);
    memcpy(&makeItem,table+4,sizeof makeItem); memcpy(&makeArray,table+5,sizeof makeArray);
    memcpy(&mixed,table+6,sizeof mixed);
    for(int i=0;i<100;i++) {
        OInt a=makeInt(1,i), none=makeInt(0,i);
        ODouble b=makeDouble(1,3.5); OFloat c=makeFloat(1,1.25f);
        OBool d=makeBool(1,1); OItem e=makeItem(1,i); OArray f=makeArray(1);
        if(a.tag || a.value!=i || none.tag!=1 || b.tag || b.value!=3.5 ||
           c.tag || c.value!=1.25f || d.tag || !d.value || e.tag ||
           e.value.count!=i || e.value.amount!=3.5 || f.tag || f.value[2]!=9) return 1;
        int calls=0; ODouble result=mixed(a,1,2.5,&calls,b,1.25f,1);
        if(result.tag || result.value!=6.0 || calls!=1) return 2;
        result=mixed(a,1,2.5,&calls,b,1.25f,0);
        if(result.tag!=1 || calls!=2) return 3;
    }
    return 0;
}

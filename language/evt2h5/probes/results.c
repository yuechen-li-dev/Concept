/* Agreed private native shapes, not the bootstrap Result compatibility ABI. */
#include <stdint.h>
#include <string.h>
#define EXPORT __declspec(dllexport)
typedef struct { int count; double amount; } Item;
typedef struct { int first, second; } Pair;
typedef struct { uint32_t tag; union { int value; uint32_t error; } payload; } RI;
typedef struct { uint32_t tag; union { double value; uint32_t error; } payload; } RD;
typedef struct { uint32_t tag; union { Item value; uint32_t error; } payload; } RR;
typedef struct { uint32_t tag; union { int value[3]; uint32_t error; } payload; } RA;
typedef struct { uint32_t tag; union { Pair value; uint16_t error; } payload; } RP;
_Static_assert(sizeof(RI)==8 && sizeof(RD)==16 && sizeof(RR)==24 && sizeof(RA)==16 && sizeof(RP)==12,"native shapes");
typedef RI (*Parse)(_Bool,int);
typedef RD (*Decimal)(_Bool,double);
typedef RR (*Record)(_Bool,int);
typedef RA (*Array)(_Bool);
typedef RP (*Pairs)(_Bool,int);
typedef RD (*Mixed)(RI,int,double,int*,RD,float,_Bool);
typedef RI (*Validate)(_Bool);
EXPORT RI ResultIntRelay(RI value){return value;}
EXPORT RD ResultDoubleRelay(RD value){return value;}
EXPORT RP ResultPairRelay(RP value){return value;}
EXPORT int CallResults(const void *const *table){
    Parse parse, doubled; Decimal decimal; Record record; Array array; Pairs pairs; Mixed mixed; Validate validate;
    memcpy(&parse,table+0,sizeof parse);memcpy(&decimal,table+1,sizeof decimal);memcpy(&record,table+2,sizeof record);memcpy(&array,table+3,sizeof array);memcpy(&pairs,table+4,sizeof pairs);memcpy(&mixed,table+5,sizeof mixed);memcpy(&validate,table+6,sizeof validate);memcpy(&doubled,table+7,sizeof doubled);
    for(int i=0;i<20;i++){
        RI a=parse(1,i), error=parse(0,i), twice=doubled(1,i), failed=doubled(0,i);RD d=decimal(1,3.5);RR r=record(1,i);RA values=array(1);RP pair=pairs(1,i),bad=pairs(0,i);RI unit=validate(1);
        if(a.tag||a.payload.value!=i||error.tag!=1||error.payload.error!=1||twice.tag||twice.payload.value!=2*i||failed.tag!=1||failed.payload.error!=1||unit.tag||d.tag||d.payload.value!=3.5||r.tag||r.payload.value.count!=i||values.tag||values.payload.value[2]!=9||pair.tag||pair.payload.value.second!=i+1||bad.tag!=1||bad.payload.error!=50000)return 1;
        int calls=0;RD output=mixed(a,1,2.5,&calls,d,1.25f,1);if(output.tag||output.payload.value!=6.0||calls!=1)return 2;
        output=mixed(error,1,2.5,&calls,d,1.25f,1);if(output.tag!=1||output.payload.error!=1||calls!=1)return 3;
    }
    return 0;
}

/* Independent Microsoft-default x64 C compiler probe. This is test input,
 * not a Cathedral classifier or generated-code execution qualification. */
typedef unsigned char u8;
typedef unsigned short u16;
typedef unsigned long long u64;
typedef struct One { u8 value; } One;
typedef struct Two { u16 value; } Two;
typedef struct Four { int value; } Four;
typedef struct Eight { u64 value; } Eight;
typedef struct Three { u8 a, b, c; } Three;
typedef struct Five { u8 values[5]; } Five;
typedef struct Six { u8 values[6]; } Six;
typedef struct Seven { u8 values[7]; } Seven;
typedef struct Twelve { int a, b, c; } Twelve;
typedef struct Sixteen { double a, b; } Sixteen;
typedef struct TwentyFour { double a, b, c; } TwentyFour;
typedef struct Padded { u8 tag; u64 value; } Padded;
typedef struct Nested { Four first, second; } Nested;
typedef struct ArrayEight { u8 values[8]; } ArrayEight;
_Static_assert(sizeof(One) == 1, "one");
_Static_assert(sizeof(Two) == 2, "two");
_Static_assert(sizeof(Four) == 4, "four");
_Static_assert(sizeof(Eight) == 8, "eight");
_Static_assert(sizeof(Three) == 3, "three");
_Static_assert(sizeof(Twelve) == 12, "twelve");
_Static_assert(sizeof(Sixteen) == 16, "sixteen");
_Static_assert(sizeof(TwentyFour) == 24, "twenty four");
_Static_assert(sizeof(Padded) == 16, "padding");
One RoundOne(One x) { return x; }
Two RoundTwo(Two x) { return x; }
Four RoundFour(Four x) { return x; }
Eight RoundEight(Eight x) { return x; }
Three RoundThree(Three x) { return x; }
Five RoundFive(Five x) { return x; }
Six RoundSix(Six x) { return x; }
Seven RoundSeven(Seven x) { return x; }
Twelve RoundTwelve(Twelve x) { return x; }
Sixteen RoundSixteen(Sixteen x) { return x; }
TwentyFour RoundTwentyFour(TwentyFour x) { return x; }
Padded RoundPadded(Padded x) { return x; }
Nested RoundNested(Nested x) { return x; }
ArrayEight RoundArray(ArrayEight x) { return x; }
TwentyFour Shifted(int a, double b, int c, float d) {
    TwentyFour result = { (double)a, b, (double)c + d };
    return result;
}
extern int Consume(TwentyFour value);
int CallerTemporary(TwentyFour value) {
    return Consume(value);
}

#ifndef __OPTIMIZE__
#error This native consumer requires optimization.
#endif
#if __cplusplus < 202100L
#error This native consumer requires C++23.
#endif
int main() { return 0; }

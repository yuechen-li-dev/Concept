#include <stdatomic.h>
#include <stddef.h>
#include <stdint.h>

typedef struct {
    float temperature[16];
    float source[16];
} HeatField;

void CAdvanceHeat(HeatField *next, const HeatField *current, float diffusion) {
    next->temperature[0] = current->temperature[0];
    for (int i = 1; i < 15; ++i) {
        float laplace = current->temperature[i - 1]
            - 2.0f * current->temperature[i]
            + current->temperature[i + 1];
        next->temperature[i] = current->temperature[i]
            + diffusion * laplace + current->source[i];
    }
    next->temperature[15] = current->temperature[15];
}

typedef struct { int symbol; int price_ticks; uint64_t timestamp; } MarketEvent;
typedef struct {
    MarketEvent slots[64];
    atomic_int head;
    atomic_int tail;
} MarketQueue;

int CPublish(MarketQueue *queue, MarketEvent event) {
    int tail = atomic_load_explicit(&queue->tail, memory_order_relaxed);
    int head = atomic_load_explicit(&queue->head, memory_order_acquire);
    if (tail - head == 64) return 0;
    queue->slots[tail % 64] = event;
    atomic_store_explicit(&queue->tail, tail + 1, memory_order_release);
    return 1;
}

typedef struct { int x; int y; int health; int intent; int target_x; int target_y; } Agent;

void CTickAgents(Agent *agents, size_t count) {
    for (size_t i = 0; i < count; ++i) {
        Agent *agent = &agents[i];
        if (agent->intent == 1) {
            if (agent->x < agent->target_x) ++agent->x;
            if (agent->y < agent->target_y) ++agent->y;
        }
    }
}

typedef struct { int tag; int value; int left; int right; } Expr;

int CEvaluateDense(const Expr *nodes, size_t count, int root, int *scratch) {
    for (size_t i = 0; i < count; ++i) {
        const Expr *node = &nodes[i];
        if (node->tag == 0) scratch[i] = node->value;
        else if (node->tag == 1) scratch[i] = scratch[node->left] + scratch[node->right];
        else scratch[i] = scratch[node->left] * scratch[node->right];
    }
    return scratch[root];
}

# Embedded serial controller

Intent: fixed-capacity UART transmit queue with explicit MMIO effects, payload commands, bounded draining, and recoverable overflow. The first draft is preserved in `first-draft.concept.txt`; it was written from embedded C instincts after reading the language and hardware guides.

First check: `CV4648: use 'and' instead of '&&'` at the C-style loop. The draft also uses C-style `for`, which the loops guide explicitly rejects; its guidance was stale about finite Concept ranges. These are intentional Concept differences, not requests for C syntax.

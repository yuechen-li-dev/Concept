# Dominatus parity oracle

There is intentionally no `Dominatus` Concept package, module, or namespace in
this repository. The sibling C# Dominatus repository is the consumer oracle used
to determine which historical DragonGod capabilities are semantically useful.
The native implementation stays entirely in the `DragonGod` package and
`DragonGod.*` modules.

The relevant C# seams are the HFSM instance/state-return model, AI event bus and
cursor, blackboard revisions and dirty keys, actuator host and pending
obligations, trace sink, replay driver, checkpoint builder, clock, agent/world
identity, and explicit seed state. Their canonical native replacements are
listed in `docs/design/DRAGONGOD-DOMINATUS-PARITY.md`.

This choice supersedes the conditional R7b suggestion to create
`libraries/Dominatus`: adding a second package would misrepresent the intended
architecture and make the native DragonGod substrate depend on an application
name. A future native application can depend on `DragonGod -> Standard` without
changing the kernel namespace.

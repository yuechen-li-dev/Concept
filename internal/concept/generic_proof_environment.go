package concept

import "sort"

// Parse and Generate analyze independently. Only the concrete consumer
// module carries its demanded imported proof entries across that boundary;
// an exporting artifact re-exports its own access summaries separately and
// must not accumulate dependency instances in its semantic payload.
func evt1MaterializeGenericProofSummaries(module *Module, env *semanticEnv) {
	seen := map[string]bool{}
	for _, entry := range module.AccessSummaries {
		seen[entry.ID] = true
	}
	for _, entry := range env.accessSummaries {
		if entry.Instance != "" && !seen[entry.ID] {
			module.AccessSummaries = append(module.AccessSummaries, entry)
			seen[entry.ID] = true
		}
	}
	evt1SortAccessEntries(module.AccessSummaries)
}

// evt1CloseGenericAccessSummaries projects imported symbolic access evidence
// into each demanded concrete generic instance. The open artifact is immutable;
// each instance receives independent entries with the original evidence origin.
// Proof status is re-derived from these concrete entries, never copied from an
// open generic proposition.
func evt1CloseGenericAccessSummaries(env *semanticEnv) error {
	identities := make([]string, 0, len(env.genericTypeApplications))
	for identity := range env.genericTypeApplications {
		identities = append(identities, identity)
	}
	sort.Strings(identities)
	open := append([]MIRAccessEntry(nil), env.accessSummaries...)
	for _, identity := range identities {
		application := env.genericTypeApplications[identity]
		generic, ok := env.genericTypes[application.Name]
		if !ok || len(generic.Parameters) != len(application.TypeArgs) {
			continue
		}
		bindings := evt1TemplateBindings(generic.Parameters, application.TypeArgs)
		for _, source := range open {
			if source.OpenGenericOwner != application.Name {
				continue
			}
			entry := source
			entry.OpenGenericOwner = ""
			entry.Instance = identity
			entry.Subject.Root.Type = evt1CloseAccessType(env, source.Subject.Root.Type, bindings)
			entry.Subject.Type = evt1CloseAccessType(env, source.Subject.Type, bindings)
			entry.Context.Type = evt1CloseAccessType(env, source.Context.Type, bindings)
			if _, symbolic := bindings[source.Context.Name]; symbolic {
				entry.Context.Name = entry.Context.Type.String()
			}
			entry.Function.Type = evt1CloseAccessType(env, source.Function.Type, bindings)
			entry.Module.Type = evt1CloseAccessType(env, source.Module.Type, bindings)
			if evt1TypeContainsConceptParameter(entry.Subject.Root.Type) || evt1TypeContainsConceptParameter(entry.Subject.Type) || evt1TypeContainsConceptParameter(entry.Context.Type) {
				return evt1Diagnostic("GENERIC_PROOF_NOT_CLOSED", "concrete generic access evidence contains an unresolved bound parameter: "+identity, source.SourceSpan)
			}
			entry = evt1FinalizeAccessEntry(entry)
			env.accessSummaries = append(env.accessSummaries, entry)
		}
	}
	return nil
}

func evt1CloseAccessType(env *semanticEnv, original Type, bindings map[string]Type) Type {
	if original.Name == "" && original.Kind == "" {
		return original
	}
	closed := evt1SubstituteBindings(original, bindings)
	if resolved, err := evt1ResolveType(env, nil, closed); err == nil {
		return resolved
	}
	return closed
}

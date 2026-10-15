package concept

// Closed template functions have translation-unit-local linkage. Validation
// can materialize instances while checking imported open bodies or compile-time
// queries which never become runtime calls. Keep only the transitive closure
// referenced by emitted runtime support and ordinary function bodies. This is
// C-backend symbol liveness, not a semantic proof or a validation shortcut.
// Scan identifiers conservatively (including comments and literals): an extra
// match can retain code, but cannot remove a possible reference. Async lowering
// owns multiple entry symbols, so its instances remain conservative roots.
func evt1LiveTemplateBodies(roots string, bodies map[string]string, async map[string]bool) map[string]bool {
	live := map[string]bool{}
	queue := []string{roots}
	for symbol := range async {
		if body, ok := bodies[symbol]; ok {
			live[symbol] = true
			queue = append(queue, body)
		}
	}
	for len(queue) > 0 {
		text := queue[0]
		queue = queue[1:]
		for i := 0; i < len(text); {
			if !evt1CIdentifierStart(text[i]) {
				i++
				continue
			}
			start := i
			for i++; i < len(text) && (evt1CIdentifierStart(text[i]) || text[i] >= '0' && text[i] <= '9'); i++ {
			}
			symbol := text[start:i]
			if body, ok := bodies[symbol]; ok && !live[symbol] {
				live[symbol] = true
				queue = append(queue, body)
			}
		}
	}
	return live
}

func evt1CIdentifierStart(c byte) bool {
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

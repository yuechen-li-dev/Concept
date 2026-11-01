package concept

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Phase qualification connects fresh checked generation to the existing native
// loops. Only immutable Cathedral builds are shared. The ordinary milestone
// harness, its trial count, and all historical byte/sentinel checks stay intact.
func phaseFreshNativeHost(t *testing.T, host string, canonical []byte, family string, produce func() []byte) string {
	t.Helper()
	if os.Getenv("CONCEPT_EVT2H_PHASE_QUALIFICATION") != "1" {
		return host
	}
	dir := t.TempDir()
	var paths strings.Builder
	paths.WriteString("static const char*phase_artifacts[100]={")
	for trial := 0; trial < 100; trial++ {
		fresh := produce()
		if !bytes.Equal(fresh, canonical) {
			t.Fatalf("fresh artifact %d differs before native admission", trial)
		}
		if _, err := DecodeMachineBridge(fresh); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, fmt.Sprintf("fresh-%03d.cmir", trial))
		if err := os.WriteFile(path, fresh, 0600); err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&paths, "%q,", filepath.ToSlash(path))
	}
	paths.WriteString("};\n")
	codec := map[string]string{"enum": "enum", "tagged": "tagged", "pattern": "pattern", "fallible": "fallible"}[family]
	if codec == "" {
		t.Fatal("unknown phase bridge family", family)
	}
	if !strings.Contains(host, "F(emit_module)(src,") || !strings.Contains(host, "int main(") {
		t.Fatal("phase hook did not find canonical emission/main")
	}
	// Replace calls before inserting the adapter, which calls the real production
	// emitter. The adapter contains only file I/O, codec and equality checks.
	host = strings.ReplaceAll(host, "F(emit_module)(src,", "phase_emit_module(src,")
	adapter := paths.String() + fmt.Sprintf(phaseFreshAdapter, codec, codec)
	host = strings.Replace(host, "int main(", adapter+"\nint main(", 1)
	t.Log("phase gate: 100 freshly checked artifact files; each native emission independently reloads, Concept-roundtrips and admits its selected input")
	return host
}

const phaseFreshAdapter = `
static unsigned char phase_input[2000000],phase_roundtrip[2000000];
static int phase_cursor,phase_count,phase_registered,phase_seen[100];
static void phase_check_complete(void){
#if defined(ENUM_NATIVE_ONLY)||defined(TAGGED_NATIVE_ONLY)||defined(MATCH_NATIVE_ONLY)||defined(OPTION_NATIVE_ONLY)||defined(RESULT_NATIVE_ONLY)||defined(FALLIBLE_NATIVE_ONLY)
 if(phase_count!=100){fprintf(stderr,"fresh native gate incomplete: %%d/100\n",phase_count);_Exit(74);}
 puts("100 distinct freshly generated artifacts roundtripped/admitted/emitted through Cathedral");
#endif
}
static concept_result_int_native_error phase_emit_module(concept_readonly_span_byte expected,concept_span_byte output,concept_native_image*image){
 if(!phase_registered){if(atexit(phase_check_complete))abort();phase_registered=1;}
 int trial=phase_cursor++%%100;FILE*f=fopen(phase_artifacts[trial],"rb");if(!f){fprintf(stderr,"missing fresh artifact %%d\n",trial);abort();}
 size_t n=fread(phase_input,1,sizeof phase_input,f);int extra=fgetc(f);fclose(f);
 if(extra!=EOF||n!=expected.length||memcmp(phase_input,expected.data,n)){fprintf(stderr,"fresh artifact %%d differs\n",trial);abort();}
 concept_readonly_span_byte src={phase_input,n};
 concept_result_int_%s_error encoded=F(validated_%s_bridge_round_trip)(src,(concept_span_byte){phase_roundtrip,sizeof phase_roundtrip});
 if(encoded.tag||encoded.payload.ok.value!=(int)n||memcmp(phase_roundtrip,phase_input,n)){fprintf(stderr,"fresh Concept codec mismatch %%d\n",trial);abort();}
 concept_result_int_native_error emitted=F(emit_module)(src,output,image);
 if(!emitted.tag&&!phase_seen[trial]){phase_seen[trial]=1;phase_count++;}
 return emitted;
}
`

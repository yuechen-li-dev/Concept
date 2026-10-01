package concept

import "testing"

const cv4138Header = "module ImmovableFields;\nprofile Core;\nimmovable struct State { int id; };\n"

func cv4138Cases() []innateAgreementCase {
	return []innateAgreementCase{
		{"plain.concept", cv4138Header + "struct Holder { State state; };\n"},
		{"array.concept", cv4138Header + "struct Holder { State<array>[2] states; };\n"},
		{"immovable.concept", cv4138Header + "immovable struct Holder { State state; };\n"},
		{"reference.concept", cv4138Header + "ref struct Holder { ref const State state; };\n"},
		{"raw.concept", cv4138Header + "struct Holder { State<raw>[2] states; };\n"},
		{"sparse.concept", cv4138Header + "struct Holder { State<sparse>[2] states; };\n"},
		{"generic.concept", cv4138Header + "template <typename T> struct Holder { T state; }\nvoid Use(ref Holder<State> holder);\n"},
	}
}

func TestCV4138InnateAgreement(t *testing.T) {
	assertInnateAgreement(t, "CV4138", append(innateAgreementCorpus(t), cv4138Cases()...))
}

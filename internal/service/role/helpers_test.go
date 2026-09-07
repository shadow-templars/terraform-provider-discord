package role

import "testing"

func TestSplitImportID(t *testing.T) {
	cases := []struct {
		id       string
		parts    int
		wantNil  bool
		wantVals []string
	}{
		{"123:456", 2, false, []string{"123", "456"}},
		{"123", 2, true, nil},
		{"123:456:789", 2, false, []string{"123", "456:789"}},
		{"", 2, true, nil},
	}
	for _, c := range cases {
		got := splitImportID(c.id, c.parts)
		if c.wantNil {
			if got != nil {
				t.Errorf("splitImportID(%q,%d) = %v, want nil", c.id, c.parts, got)
			}
			continue
		}
		if len(got) != len(c.wantVals) {
			t.Fatalf("splitImportID(%q,%d) = %v, want %v", c.id, c.parts, got, c.wantVals)
		}
		for i := range got {
			if got[i] != c.wantVals[i] {
				t.Errorf("splitImportID(%q,%d)[%d] = %q, want %q", c.id, c.parts, i, got[i], c.wantVals[i])
			}
		}
	}
}

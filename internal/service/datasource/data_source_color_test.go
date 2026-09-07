package datasource

import "testing"

func TestParseHexColor(t *testing.T) {
	cases := []struct {
		in      string
		want    int64
		wantErr bool
	}{
		{"#5b2c8e", 5975182, false},
		{"5b2c8e", 5975182, false},
		{"ffffff", 16777215, false},
		{"000000", 0, false},
		{"nothex", 0, true},
		{"", 0, true},
	}
	for _, c := range cases {
		got, err := parseHexColor(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("parseHexColor(%q): want error, got nil", c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseHexColor(%q): unexpected error %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("parseHexColor(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

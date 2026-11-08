package integrations

import "testing"

func TestAppURLUsesFirstOrigin(t *testing.T) {
	cases := map[string]string{
		"http://localhost:3000,https://www.moonveils.xyz,https://moonveils.xyz": "http://localhost:3000",
		" https://www.moonveils.xyz/ ":                                          "https://www.moonveils.xyz",
		"https://www.moonveils.xyz":                                             "https://www.moonveils.xyz",
	}
	for in, want := range cases {
		if got := (&Secrets{FrontendURL: in}).AppURL(); got != want {
			t.Errorf("AppURL(%q) = %q, want %q", in, got, want)
		}
	}
}

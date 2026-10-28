package cmd

import "testing"

func TestIsPublicRoute(t *testing.T) {
	cases := map[string]bool{
		"/pmta/abc.def":                    true,
		"/roundcube":                       true,
		"/roundcube/?_task=login":          true,
		"/.well-known/acme-challenge/tok":  true,
		"/landing/video":                   true,
		"/pmta/../api/settings":            false,
		"/roundcube/../api/x":              false,
		"/api/settings":                    false,
		"/":                                false,
		"/pmtafake":                        false,
		"/roundcubex":                      false,
		"/landing/video/../../api/account": false,
	}

	for path, want := range cases {
		if got := isPublicRoute(path); got != want {
			t.Errorf("isPublicRoute(%q) = %v, want %v", path, got, want)
		}
	}
}

package primitives

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestKeycapBadge(t *testing.T) {
	verbs := []string{"GET", "POST", "PUT", "PATCH", "DEL", "GQL", "WSS", "SSE"}

	for _, verb := range verbs {
		t.Run(verb, func(t *testing.T) {
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Row(c).Children(func() {
					KeycapBadge(c, verb)
				})
			}, 200, 100)
			tt.Frame()

			if !tt.HasText(verb) {
				t.Fatalf("expected keycap badge to contain text %q", verb)
			}
		})
	}
}

func TestStatusCodeBadge(t *testing.T) {
	cases := []struct {
		code int
		text string
		want string
	}{
		{200, "OK", "200 OK"},
		{201, "CREATED", "201 CREATED"},
		{429, "LIMIT", "429 LIMIT"},
		{502, "ERR", "502 ERR"},
	}

	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Row(c).Children(func() {
					StatusCodeBadge(c, tc.code, tc.text)
				})
			}, 300, 100)
			tt.Frame()

			if !tt.HasText(tc.want) {
				t.Fatalf("expected status badge to contain text %q", tc.want)
			}
		})
	}
}

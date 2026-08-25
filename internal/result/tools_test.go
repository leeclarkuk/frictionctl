package result

import "testing"

func TestCountToolTransitions(t *testing.T) {
	cases := []struct {
		tools []string
		want  int
	}{
		{nil, 0},
		{[]string{"demo-platform"}, 0},
		{[]string{"demo-platform", "demo-platform", "demo-platform"}, 0},
		{[]string{"demo-platform", "demo-kube"}, 1},
		{[]string{"demo-platform", "demo-kube", "demo-platform"}, 2},
		{[]string{"", "demo-platform", "", "demo-platform"}, 0},
		{[]string{"demo-platform", "", "demo-kube", "demo-platform"}, 2},
	}
	for _, tc := range cases {
		if got := CountToolTransitions(tc.tools); got != tc.want {
			t.Fatalf("CountToolTransitions(%v) = %d, want %d", tc.tools, got, tc.want)
		}
	}
}

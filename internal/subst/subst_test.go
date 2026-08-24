package subst

import "testing"

func TestExpandAndRequire(t *testing.T) {
	vars := map[string]string{"WORKDIR": "/tmp/work"}
	got := Expand("out=${WORKDIR}/svc", vars)
	if got != "out=/tmp/work/svc" {
		t.Fatalf("got %q", got)
	}
	all := ExpandAll([]string{"${WORKDIR}", "ok"}, vars)
	if all[0] != "/tmp/work" || all[1] != "ok" {
		t.Fatalf("got %#v", all)
	}
	if err := RequireExpanded("command", []string{"${MISSING}"}); err == nil {
		t.Fatal("expected unresolved error")
	}
}

func TestMerge(t *testing.T) {
	got := Merge(map[string]string{"A": "1", "B": "2"}, map[string]string{"B": "3"})
	if got["A"] != "1" || got["B"] != "3" {
		t.Fatalf("got %#v", got)
	}
}

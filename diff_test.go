package diff

import (
	"io"
	"reflect"
	"testing"
)

func TestUpstreamPathing(t *testing.T) {
	if len(upstreamPaths("")) != 2 {
		t.Errorf(`Expected two things from "", got %#v`, upstreamPaths(""))
	}
	m := map[string]bool{}
	for _, p := range upstreamPaths("/a/b/c/d/e/f") {
		m[p] = true
	}
	if len(m) != 7 {
		t.Errorf("Expected 7 elements, got %v", m)
	}
	for _, e := range []string{"", "/a", "/a/b", "/a/b/c", "/a/b/c/d", "/a/b/c/d/e"} {
		if !m[e] {
			t.Errorf("Expected %q, but didn't find it", e)
		}
	}
}

func TestDiffNil(t *testing.T) {
	diffs, err := JSON(nil, nil)
	if err == nil {
		t.Fatalf("Expected error on nil diff, got %v", diffs)
	}
}

func TestDiffNames(t *testing.T) {
	tests := map[Type]string{
		Same:      "same",
		MissingA:  "missing a",
		MissingB:  "missing b",
		Different: "different",
	}

	for k, v := range tests {
		if k.String() != v {
			t.Errorf("Expected %v for %d, got %v", v, k, k)
		}
	}
}

func TestMust(t *testing.T) {
	must(nil) // no panic
	panicked := false
	func() {
		defer func() { panicked = recover() != nil }()
		must(io.EOF)
	}()
	if !panicked {
		t.Fatalf("Expected a panic, but didn't get one")
	}
}

func TestDiff(t *testing.T) {
	var (
		aFirst = `{"a": 1, "b": 3.2}`
		bFirst = `{"b":3.2,"a":1}`
		aTwo   = `{"a": 2, "b": 3.2}`
		aOnly1 = `{"a": 1}`
		aOnly3 = `{"a": 3}`
		broken = `{x}`
		ax1    = `{"a": {"x": 1}}`
		ax2    = `{"a": {"x": 2}}`
		esc1   = `{"a": {"/": 1}}`
		esc2   = `{"a": {"/": 2}}`
	)

	empty := map[string]Type{}
	// Interesting side-effect, in an empty map, all look same
	if empty["/a/b/c"] != Same {
		t.Errorf("Expected same in empty map lookup, got %v", empty["/a/b/c"])
	}

	tests := []struct {
		name    string
		a, b    string
		exp     map[string]Type
		errored bool
	}{
		{"Empty", "", "", empty, true},
		{"Identity", aFirst, aFirst, empty, false},
		{"Same", aFirst, bFirst, empty, false},
		{"Other order", aFirst, bFirst, empty, false},
		{"A diff", aFirst, aTwo, map[string]Type{"/a": Different}, false},
		{"A diff rev", aTwo, aFirst, map[string]Type{"/a": Different}, false},
		{"Missing b <- 1", aFirst, aOnly1, map[string]Type{"/b": MissingB}, false},
		{"Missing b -> 1", aOnly1, aFirst, map[string]Type{"/b": MissingA}, false},
		{"Missing b <- 3", aTwo, aOnly3, map[string]Type{
			"/a": Different,
			"/b": MissingB,
		}, false},
		{"Missing b -> 3", aOnly3, aTwo, map[string]Type{
			"/a": Different,
			"/b": MissingA,
		}, false},
		{"Broken A", broken, aFirst, nil, true},
		{"Broken B", aFirst, broken, nil, true},
		{"/a/x same", ax1, ax1, empty, false},
		{"/a/x different", ax1, ax2, map[string]Type{
			"/a/x": Different,
		}, false},
		{"/a/~1 different", esc1, esc2, map[string]Type{
			"/a/~1": Different,
		}, false},
	}

	for _, test := range tests {
		diffs, err := JSON([]byte(test.a), []byte(test.b))
		if (err != nil) != test.errored {
			t.Errorf("Expected error=%v on %q:  %v", test.errored, test.name, err)
		}
		if err != nil {
			continue
		}
		if !reflect.DeepEqual(test.exp, diffs) {
			t.Errorf("Unexpected diff for %q: %v\nwanted %v",
				test.name, diffs, test.exp)
		}
	}

}

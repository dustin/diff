package diff

import (
	"log"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/dustin/go-jsonpointer"
)

// DiffType represents the type of difference between two JSON
// structures.
type DiffType int

const (
	// MissingA designates a path that was missing from the first
	// argument of the diff.
	MissingA = DiffType(iota)
	// MissingB designates a path taht was missing from the second
	// argument of the diff.
	MissingB
	// Different designates a path that is found in both
	// arguments, but with different values.
	Different
)

var diffNames = []string{"missing a", "missing b", "different"}

func (d DiffType) String() string {
	return diffNames[d]
}

func must(err error) {
	if err != nil {
		log.Panic(err)
	}
}

func pointerSet(j []byte) (map[string]bool, error) {
	a, err := jsonpointer.ListPointers(j)
	if err != nil {
		return nil, err
	}
	rv := map[string]bool{}
	for _, v := range a {
		rv[v] = true
	}
	return rv, nil
}

// lengthSorting is a string slice sort.Interface that sorts longer
// strings first.
type lengthSorting []string

func (l lengthSorting) Len() int           { return len(l) }
func (l lengthSorting) Less(i, j int) bool { return len(l[j]) < len(l[i]) }
func (l lengthSorting) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }

func upstreamPaths(u string) []string {
	rv := []string{""}
	prev := "/"
	for _, p := range strings.Split(u, "/") {
		rv = append(rv, prev)
		prev = filepath.Join(prev, p)
	}

	return rv
}

// JSON returns the differences between two json blobs.
func JSON(a, b []byte) (map[string]DiffType, error) {
	amap, err := pointerSet(a)
	if err != nil {
		return nil, err
	}
	bmap, err := pointerSet(b)
	if err != nil {
		return nil, err
	}

	rv := map[string]DiffType{}
	var common lengthSorting
	for v := range amap {
		if bmap[v] {
			common = append(common, v)
		} else {
			rv[v] = MissingB
		}
	}

	for v := range bmap {
		if !amap[v] {
			rv[v] = MissingA
		}
	}

	upstream := map[string]bool{}
	sort.Sort(common)
	for _, v := range common {
		if upstream[v] {
			continue
		}
		for _, u := range upstreamPaths(v) {
			upstream[u] = true
		}

		var aval, bval interface{}
		must(jsonpointer.FindDecode(a, v, &aval))
		must(jsonpointer.FindDecode(b, v, &bval))
		if !reflect.DeepEqual(aval, bval) {
			rv[v] = Different
		}
	}

	return rv, nil
}

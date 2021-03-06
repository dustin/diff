package diff

import (
	"log"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/dustin/go-jsonpointer"
)

// Type represents the type of difference between two JSON structures.
type Type int

const (
	// Same designates a path yields the same results in two objects.
	Same = Type(iota)
	// MissingA designates a path that was missing from the first
	// argument of the diff.
	MissingA
	// MissingB designates a path taht was missing from the second
	// argument of the diff.
	MissingB
	// DifferentValue designates a path that is found in both
	// arguments, but with different values.
	DifferentValue
)

var diffNames = []string{"same", "missing a", "missing b", "different value"}

func (d Type) String() string {
	return diffNames[d]
}

// Missing is true if the Type represents a value missing in either set.
func (d Type) Missing() bool {
	return d == MissingA || d == MissingB
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
func JSON(a, b []byte) (map[string]Type, error) {
	amap, err := pointerSet(a)
	if err != nil {
		return nil, err
	}
	bmap, err := pointerSet(b)
	if err != nil {
		return nil, err
	}

	// Compute a - b and a ∩ b
	rv := map[string]Type{}
	var common []string
	for v := range amap {
		if bmap[v] {
			common = append(common, v)
		} else {
			rv[v] = MissingB
		}
	}

	// Compute b - a
	for v := range bmap {
		if !amap[v] {
			rv[v] = MissingA
		}
	}

	// Find only the longest paths of a ∩ b and verify they are
	// the same.  e.g. if /x/y/z is different between a and b,
	// then only consider /x/y/z, not /x/y or /x or / or ""
	upstream := map[string]bool{}
	sort.Slice(common, func(i, j int) bool { return len(common[j]) < len(common[i]) })
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
			rv[v] = DifferentValue
		}
	}

	return rv, nil
}

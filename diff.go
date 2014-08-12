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
	// Different designates a path that is found in both
	// arguments, but with different values.
	Different
)

var diffNames = []string{"same", "missing a", "missing b", "different"}

func (d Type) String() string {
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
	var common lengthSorting
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

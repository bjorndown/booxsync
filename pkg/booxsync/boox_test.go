package booxsync

import (
	"fmt"
	"testing"
)

func buildLibrary() *BooxLibrary {
	file := BooxFile{Name: "test3.pdf"}

	test2 := BooxFile{Name: "test2", IsDir: true, Id: "2",
		Children: []*BooxFile{&file}}

	test := BooxFile{Name: "test", IsDir: true, Id: "1",
		Children: []*BooxFile{&test2}}

	root := BooxFile{
		Children: []*BooxFile{&test},
		IsDir:    true,
	}
	l := BooxLibrary{Root: &root}

	file.Parent = &test2
	test2.Parent = &test
	test.Parent = &root
	return &l
}

func TestBooxLibrary_Exists(t *testing.T) {
	l := buildLibrary()

	cases := []struct {
		path      string
		mustExist bool
	}{
		{"test/test2/test3.pdf", true},
		{"test/test2/", true},
		{"test/test2/nope.pdf", false},
		{"test/nope/", false},
		{"test2", false},
		{".", true},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("must return %t if file exists", c.mustExist), func(t *testing.T) {
			if exists, _ := l.Exists(c.path); exists != c.mustExist {
				t.Errorf("should return %t for path %q", c.mustExist, c.path)
			}
		})
	}
}

func TestBooxLibrary_GetParentId(t *testing.T) {
	l := buildLibrary()

	cases := []struct {
		path            string
		id              string
		mustReturnError bool
	}{
		{"test/test2/test3.pdf", "2", false},
		{"test/test2/", "1", false},
		{"test/test2/nope.pdf", "", true},
		{"test/nope/", "", true},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("must return %q", c.id), func(t *testing.T) {
			id, err := l.GetParentId(c.path)
			if err != nil && !c.mustReturnError {
				t.Errorf("unexpected error: %s", err)
			}
			if id != c.id {
				t.Errorf("must return %q for path %q, but returned %q", c.id, c.path, id)
			}
		})
	}
}

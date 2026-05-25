package services

import (
	"testing"
)

func TestValidateSkillSegment(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "valid kebab-case", input: "documentation-writer", wantErr: false},
		{name: "empty string", input: "", wantErr: true},
		{name: "dot", input: ".", wantErr: true},
		{name: "double dot", input: "..", wantErr: true},
		{name: "absolute path", input: "/abs", wantErr: true},
		{name: "contains slash", input: "a/b", wantErr: true},
		{name: "traversal a/../b", input: "a/../b", wantErr: true},
		{name: "dot-slash prefix", input: "./a", wantErr: true},
		{name: "trailing slash", input: "a/", wantErr: true},
		{name: "NUL byte", input: "a\x00b", wantErr: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateSkillSegment(tc.input)
			if tc.wantErr && err == nil {
				t.Errorf("validateSkillSegment(%q): expected error, got nil", tc.input)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("validateSkillSegment(%q): expected nil, got %v", tc.input, err)
			}
		})
	}
}

func TestIsWithin(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		root string
		path string
		want bool
	}{
		{name: "direct child", root: "/foo", path: "/foo/bar", want: true},
		{name: "nested child", root: "/foo", path: "/foo/bar/baz", want: true},
		{name: "outside root", root: "/foo", path: "/etc", want: false},
		{name: "root itself", root: "/foo", path: "/foo", want: false},
		{name: "sibling prefix", root: "/foo", path: "/foobar", want: false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := isWithin(tc.root, tc.path)
			if got != tc.want {
				t.Errorf("isWithin(%q, %q) = %v, want %v", tc.root, tc.path, got, tc.want)
			}
		})
	}
}

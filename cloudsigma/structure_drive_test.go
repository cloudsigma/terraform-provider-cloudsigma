package cloudsigma

import (
	"reflect"
	"testing"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

func TestStructureDrive_expandTags(t *testing.T) {
	cases := []struct {
		description string
		input       []interface{}
		expected    []cloudsigma.Tag
	}{
		{"Nil", nil, []cloudsigma.Tag{}},
		{"SingleTag",
			[]interface{}{"single-tag-uuid"},
			[]cloudsigma.Tag{{UUID: "single-tag-uuid"}},
		},
		{
			"MultipleTags",
			[]interface{}{"first-tag-uuid", "second-tag-uuid"},
			[]cloudsigma.Tag{
				{UUID: "first-tag-uuid"},
				{UUID: "second-tag-uuid"},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			got := expandTags(c.input)
			if len(got) != len(c.expected) {
				t.Fatalf("expected length: %#v: got: %#v", len(got), len(c.expected))
			}
			if !reflect.DeepEqual(got, c.expected) {
				t.Fatalf("expected: %#v, got: %#v", c.expected, got)
			}
		})
	}
}

func TestStructureDrive_flattenTags(t *testing.T) {
	cases := []struct {
		description string
		input       []cloudsigma.Tag
		expected    []interface{}
	}{
		{"Nil", nil, []interface{}{}},
		{"SingleTag",
			[]cloudsigma.Tag{{UUID: "single-tag-uuid"}},
			[]interface{}{"single-tag-uuid"},
		},
		{"MultipleTags",
			[]cloudsigma.Tag{
				{UUID: "first-tag-uuid"},
				{UUID: "second-tag-uuid"},
			},
			[]interface{}{"first-tag-uuid", "second-tag-uuid"},
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			got := flattenTags(c.input)
			if !reflect.DeepEqual(got, c.expected) {
				t.Fatalf("expected: %#v, got: %#v", c.expected, got)
			}
		})
	}
}

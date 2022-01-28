package cloudsigma

import (
	"reflect"
	"testing"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

func TestStructureServer_expandEnclavePageCaches(t *testing.T) {
	cases := []struct {
		description string
		input       []interface{}
		expected    []cloudsigma.EnclavePageCache
	}{
		{"Nil", nil, []cloudsigma.EnclavePageCache{}},
		{
			"SingleEPC",
			[]interface{}{"100M"},
			[]cloudsigma.EnclavePageCache{{Size: "100M"}},
		},
		{
			"MultipleEPCs",
			[]interface{}{"100M", "200M"},
			[]cloudsigma.EnclavePageCache{
				{Size: "100M"},
				{Size: "200M"},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			got := expandEnclavePageCaches(c.input)
			if len(got) != len(c.expected) {
				t.Fatalf("expected length: %#v: got: %#v", len(got), len(c.expected))
			}
			if !reflect.DeepEqual(got, c.expected) {
				t.Fatalf("expected: %#v, got: %#v", c.expected, got)
			}
		})
	}
}

func TestStructureServer_flattenEnclavePageCaches(t *testing.T) {
	cases := []struct {
		description string
		input       []cloudsigma.EnclavePageCache
		expected    []interface{}
	}{
		{"Nil", nil, []interface{}{}},
		{
			"SingleEPC",
			[]cloudsigma.EnclavePageCache{{Size: "100M"}},
			[]interface{}{"100M"},
		},
		{
			"MultipleEPCs",
			[]cloudsigma.EnclavePageCache{
				{Size: "100M"},
				{Size: "200M"},
			},
			[]interface{}{"100M", "200M"},
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			got := flattenEnclavePageCaches(c.input)
			if len(got) != len(c.expected) {
				t.Fatalf("expected length: %#v: got: %#v", len(got), len(c.expected))
			}
			if !reflect.DeepEqual(got, c.expected) {
				t.Fatalf("expected: %#v, got: %#v", c.expected, got)
			}
		})
	}
}

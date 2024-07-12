package migration

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestFilterApplyFilter(t *testing.T) {
	filters := []FilterModel{{
		Name: types.StringValue("id"),
		Values: []types.String{
			types.StringValue("12345"),
		},
	}}
	type FilterableEntity struct {
		ID string `json:"id"`
	}
	entities := []FilterableEntity{
		{ID: "abc"},
		{ID: "12345"},
	}
	entitiesForFilter := make([]any, len(entities))
	for i, entity := range entities {
		entitiesForFilter[i] = entity
	}

	actualResult, diags := ApplyFilter(filters, entitiesForFilter)

	assert.Nil(t, diags)
	assert.Equal(t, entities[1], actualResult[0])
}

func TestFilter_resolveStructValue(t *testing.T) {
	type TestStruct struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}
	testStruct := TestStruct{Name: "foobar"}

	actualValue, diags := resolveStructValue(testStruct, "name")

	assert.Nil(t, diags)
	assert.Equal(t, "foobar", actualValue)
}

func TestFilter_normalizeValue(t *testing.T) {
	type testCase struct {
		Input          any
		ExpectedOutput string
	}
	tests := map[string]testCase{
		"Int": {
			Input:          123,
			ExpectedOutput: "123",
		},
		"Float": {
			Input:          123.888,
			ExpectedOutput: "124",
		},
		"Bool": {
			Input:          true,
			ExpectedOutput: "true",
		},
		"String": {
			Input:          "foobar",
			ExpectedOutput: "foobar",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			actualValue, diags := normalizeValue(test.Input)

			assert.Nil(t, diags)
			assert.Equal(t, test.ExpectedOutput, actualValue)
		})
	}
}

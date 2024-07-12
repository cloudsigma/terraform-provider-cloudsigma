package migration

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type FilterModel struct {
	Name   types.String   `tfsdk:"name"`
	Values []types.String `tfsdk:"values"`
}

func ApplyFilter(filters []FilterModel, entries []any) ([]any, diag.Diagnostic) {
	result := make([]any, 0)

	for _, entry := range entries {
		matched, diags := matchesFilters(filters, entry)
		if diags != nil {
			return nil, diags
		}
		if !matched {
			continue
		}
		result = append(result, entry)
	}

	return result, nil
}

func matchesFilters(filters []FilterModel, entry any) (bool, diag.Diagnostic) {
	for _, filter := range filters {
		filterName := filter.Name.ValueString()

		// get the field from the input struct
		field, diags := resolveStructValue(entry, filterName)
		if diags != nil {
			return false, diags
		}

		// check if field matches the filter
		matched, diags := checkFieldMatchesFilter(field, filter)
		if diags != nil {
			return false, diags
		}
		if !matched {
			return false, nil
		}
	}
	return true, nil
}

func resolveStructValue(value any, field string) (any, diag.Diagnostic) {
	structField, diags := resolveStructField(value, field)
	if diags != nil {
		return nil, diags
	}

	namedField := reflect.ValueOf(value).FieldByName(structField.Name)

	if !namedField.IsValid() {
		return nil, diag.NewErrorDiagnostic(
			"Field not found",
			fmt.Sprintf("Could not find tag in struct: %s", field),
		)
	}

	return namedField.Interface(), nil
}

func resolveStructField(value any, field string) (reflect.StructField, diag.Diagnostic) {
	rType := reflect.TypeOf(value)

	for i := 0; i < rType.NumField(); i++ {
		currentField := rType.Field(i)
		if field == strings.ToLower(currentField.Name) {
			return currentField, nil
		}
	}
	return reflect.StructField{}, diag.NewErrorDiagnostic(
		"Missing field",
		fmt.Sprintf("Unable to find field '%s' in struct. Info: %s, %s", field, rType, value),
	)
}

func checkFieldMatchesFilter(field any, filter FilterModel) (bool, diag.Diagnostic) {
	rField := reflect.ValueOf(field)

	// recursively check on lists
	if rField.Kind() == reflect.Slice {
		for i := 0; i < rField.Len(); i++ {
			matched, diags := checkFieldMatchesFilter(rField.Index(i).Interface(), filter)
			if diags != nil {
				return false, diags
			}
			if matched {
				return true, nil
			}
		}
		return false, nil
	}

	normalizedValue, diags := normalizeValue(field)
	if diags != nil {
		return false, diags
	}

	for _, value := range filter.Values {
		if reflect.DeepEqual(normalizedValue, value.ValueString()) {
			return true, nil
		}
	}

	return false, nil
}

func normalizeValue(field any) (string, diag.Diagnostic) {
	rField := reflect.ValueOf(field)

	// special case for pointers
	for rField.Kind() == reflect.Pointer {
		if rField.IsNil() {
			return "", nil
		}
		rField = reflect.Indirect(rField)
	}

	switch rField.Kind() {
	case reflect.String:
		return rField.String(), nil
	case reflect.Int, reflect.Int64:
		return strconv.FormatInt(rField.Int(), 10), nil
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(rField.Float(), 'f', 0, 64), nil
	case reflect.Bool:
		return strconv.FormatBool(rField.Bool()), nil
	default:
		return "", diag.NewErrorDiagnostic(
			"Invalid field type",
			fmt.Sprintf("Got %s", rField.Type().String()),
		)
	}
}

package migration

func ToAnySlice[T any](entries []T) []any {
	result := make([]any, len(entries))

	for i, entry := range entries {
		result[i] = entry
	}

	return result
}

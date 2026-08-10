package role

import "strings"

func boolPtr(v bool) *bool {
	return &v
}

func intPtr(v int) *int {
	return &v
}

// splitImportID splits a composite import ID by ":" and returns nil if the
// number of parts doesn't match the expected count.
func splitImportID(id string, expectedParts int) []string {
	parts := strings.SplitN(id, ":", expectedParts)
	if len(parts) != expectedParts {
		return nil
	}
	return parts
}

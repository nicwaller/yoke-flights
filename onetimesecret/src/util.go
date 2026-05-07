package main

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func ptr[T any](v T) *T { return &v }

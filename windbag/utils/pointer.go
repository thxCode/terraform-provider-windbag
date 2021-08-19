package utils

// DeepCopyStringPointer returns a new pointer of the given string pointer.
func DeepCopyStringPointer(s *string) *string {
	return StringPointer(*s)
}

// StringPointer returns the pointer of the given string.
func StringPointer(s string) *string {
	return &s
}

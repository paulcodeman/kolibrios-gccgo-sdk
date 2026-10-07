package filepath

// KolibriOS paths have no Windows drive/UNC volume prefix.
func volumeNameLen(path string) int { return 0 }

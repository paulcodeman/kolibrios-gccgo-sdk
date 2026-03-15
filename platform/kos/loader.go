package kos

func LoaderParameters() string {
	return CStringToStringRaw(LoaderParametersRaw())
}

func LoaderPath() string {
	return normalizeLoaderPath(CStringToStringRaw(LoaderPathRaw()))
}

func normalizeLoaderPath(path string) string {
	if len(path) >= 3 && path[0] == '/' && path[2] == '/' {
		switch path[1] {
		case byte(EncodingCP866), byte(EncodingUTF16LE), byte(EncodingUTF8):
			return path[2:]
		}
	}
	if len(path) >= 2 {
		switch path[0] {
		case byte(EncodingCP866), byte(EncodingUTF16LE), byte(EncodingUTF8):
			return path[1:]
		}
	}

	return path
}

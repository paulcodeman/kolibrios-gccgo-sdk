package sourcemap

// Consumer is a stub sourcemap consumer for KolibriOS builds.
type Consumer struct{}

// Parse returns a nil consumer with no error to indicate that sourcemaps are unsupported.
func Parse(filename string, b []byte) (*Consumer, error) {
	return nil, nil
}

// Source reports no mapping information.
func (c *Consumer) Source(genLine, genCol int) (source, name string, line, col int, ok bool) {
	return "", "", 0, 0, false
}

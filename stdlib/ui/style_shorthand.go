package ui

func expandBoxShorthand(values []int) (top int, right int, bottom int, left int) {
	switch len(values) {
	case 1:
		top = values[0]
		right = values[0]
		bottom = values[0]
		left = values[0]
	case 2:
		top = values[0]
		right = values[1]
		bottom = values[0]
		left = values[1]
	case 3:
		top = values[0]
		right = values[1]
		bottom = values[2]
		left = values[1]
	default:
		top = values[0]
		right = values[1]
		bottom = values[2]
		left = values[3]
	}
	return
}

func expandCornerShorthand(values []int) (topLeft int, topRight int, bottomRight int, bottomLeft int) {
	switch len(values) {
	case 1:
		topLeft = values[0]
		topRight = values[0]
		bottomRight = values[0]
		bottomLeft = values[0]
	case 2:
		topLeft = values[0]
		topRight = values[1]
		bottomRight = values[0]
		bottomLeft = values[1]
	case 3:
		topLeft = values[0]
		topRight = values[1]
		bottomRight = values[2]
		bottomLeft = values[1]
	default:
		topLeft = values[0]
		topRight = values[1]
		bottomRight = values[2]
		bottomLeft = values[3]
	}
	return
}

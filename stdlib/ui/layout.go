package ui

func effectivePosition(style Style) PositionMode {
	position := PositionStatic
	if resolved, ok := resolvePosition(style.Position); ok {
		position = resolved
	}
	if position == PositionStatic {
		if style.Left != nil || style.Top != nil || style.Right != nil || style.Bottom != nil {
			position = PositionAbsolute
		}
	}
	return position
}

func resolveRect(base Rect, container Rect, style Style) Rect {
	rect := base
	width := rect.Width
	height := rect.Height
	widthSet := false
	if resolved, ok := resolveLength(style.Width); ok {
		width = resolved
		widthSet = true
	}
	heightSet := false
	if resolved, ok := resolveLength(style.Height); ok {
		height = resolved
		heightSet = true
	}

	position := effectivePosition(style)
	switch position {
	case PositionAbsolute:
		leftValue, leftSet := resolveLength(style.Left)
		rightValue, rightSet := resolveLength(style.Right)
		topValue, topSet := resolveLength(style.Top)
		bottomValue, bottomSet := resolveLength(style.Bottom)

		if leftSet && rightSet && !widthSet {
			width = container.Width - leftValue - rightValue
			if width < 0 {
				width = 0
			}
		}
		if topSet && bottomSet && !heightSet {
			height = container.Height - topValue - bottomValue
			if height < 0 {
				height = 0
			}
		}

		x := rect.X
		y := rect.Y
		if leftSet {
			x = container.X + leftValue
		} else if rightSet {
			x = container.X + container.Width - rightValue - width
		}
		if topSet {
			y = container.Y + topValue
		} else if bottomSet {
			y = container.Y + container.Height - bottomValue - height
		}

		rect = Rect{X: x, Y: y, Width: width, Height: height}
	case PositionRelative:
		rect.Width = width
		rect.Height = height
		if value, ok := resolveLength(style.Left); ok {
			rect.X += value
		}
		if value, ok := resolveLength(style.Right); ok {
			rect.X -= value
		}
		if value, ok := resolveLength(style.Top); ok {
			rect.Y += value
		}
		if value, ok := resolveLength(style.Bottom); ok {
			rect.Y -= value
		}
	default:
		rect.Width = width
		rect.Height = height
	}

	return rect
}

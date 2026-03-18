package ui

func effectivePosition(style Style) PositionMode {
	position := PositionStatic
	if resolved, ok := resolvePosition(style.position); ok {
		position = resolved
	}
	if position == PositionStatic {
		if style.left != nil || style.top != nil || style.right != nil || style.bottom != nil {
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
	if resolved, ok := resolveLength(style.width); ok {
		width = resolved
		widthSet = true
	}
	heightSet := false
	if resolved, ok := resolveLength(style.height); ok {
		height = resolved
		heightSet = true
	}

	position := effectivePosition(style)
	switch position {
	case PositionAbsolute:
		leftValue, leftSet := resolveLength(style.left)
		rightValue, rightSet := resolveLength(style.right)
		topValue, topSet := resolveLength(style.top)
		bottomValue, bottomSet := resolveLength(style.bottom)

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
		if value, ok := resolveLength(style.left); ok {
			rect.X += value
		}
		if value, ok := resolveLength(style.right); ok {
			rect.X -= value
		}
		if value, ok := resolveLength(style.top); ok {
			rect.Y += value
		}
		if value, ok := resolveLength(style.bottom); ok {
			rect.Y -= value
		}
	default:
		rect.Width = width
		rect.Height = height
	}

	return rect
}

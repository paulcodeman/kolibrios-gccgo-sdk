package ui

import surfacepkg "surface"

type Rect = surfacepkg.Rect

func UnionRect(a Rect, b Rect) Rect {
	return surfacepkg.UnionRect(a, b)
}

func IntersectRect(a Rect, b Rect) Rect {
	return surfacepkg.IntersectRect(a, b)
}

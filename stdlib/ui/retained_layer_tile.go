package ui

const (
	retainedLayerTileSize    = 256
	retainedLayerTileMinArea = 262144
)

func useRetainedLayerTiles(bounds Rect) bool {
	if bounds.Empty() || bounds.Width <= 0 || bounds.Height <= 0 {
		return false
	}
	if bounds.Width <= retainedLayerTileSize && bounds.Height <= retainedLayerTileSize {
		return false
	}
	if bounds.Width > retainedLayerTileSize*2 || bounds.Height > retainedLayerTileSize*2 {
		return true
	}
	return bounds.Width*bounds.Height >= retainedLayerTileMinArea
}

func retainedLayerTileCount(size int) int {
	if size <= 0 {
		return 0
	}
	return (size + retainedLayerTileSize - 1) / retainedLayerTileSize
}

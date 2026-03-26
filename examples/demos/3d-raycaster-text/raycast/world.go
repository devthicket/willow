package raycast

// World holds the grid map and tileset configuration for a raycaster level.
type World struct {
	Grid      [][]int // Grid[row][col]; 0 = empty, 1–15 = wall type
	Width     int     // number of columns
	Height    int     // number of rows
	WallTiles []int   // atlas tile index per wall type (index 0 unused)
	FloorTile int     // tileset index for floor texture
	CeilTile  int     // tileset index for ceiling texture
	TileSize  int     // pixels per tile in the tileset (e.g. 32)
	TileCols  int     // number of columns in the tileset grid (e.g. 4)
}

// InBounds returns true if the grid coordinate (x, y) is within the map.
func (w *World) InBounds(x, y int) bool {
	return x >= 0 && x < w.Width && y >= 0 && y < w.Height
}

// CellAt returns the wall type at grid coordinate (x, y), or 0 if out of bounds.
func (w *World) CellAt(x, y int) int {
	if !w.InBounds(x, y) {
		return 0
	}
	return w.Grid[y][x]
}

// TileUV returns the top-left pixel coordinate of a tile in the tileset.
func (w *World) TileUV(index int) (float32, float32) {
	col := index % w.TileCols
	row := index / w.TileCols
	return float32(col * w.TileSize), float32(row * w.TileSize)
}

// Walkable returns true if the grid cell at (x, y) is empty and in bounds.
// Suitable as the callback for Camera.TryMove.
func (w *World) Walkable(x, y int) bool {
	return w.InBounds(x, y) && w.CellAt(x, y) == 0
}

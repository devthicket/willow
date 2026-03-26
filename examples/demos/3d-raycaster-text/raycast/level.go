package raycast

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

// Level is the JSON-serialisable representation of a raycaster level file.
// The grid is an array of strings where each character is a hex digit (0–F).
// '0' is empty space; '1'–'F' are wall types mapped via wallTiles.
type Level struct {
	Name    string       `json:"name"`
	Spawn   LevelSpawn   `json:"spawn"`
	Tileset LevelTileset `json:"tileset"`
	Grid    []string     `json:"grid"`
}

// LevelSpawn defines the player's starting position and facing direction.
type LevelSpawn struct {
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	DirX float64 `json:"dirX"`
	DirY float64 `json:"dirY"`
}

// LevelTileset holds tileset layout and tile index mappings.
type LevelTileset struct {
	TileSize  int            `json:"tileSize"`
	TileCols  int            `json:"tileCols"`
	FloorTile int            `json:"floorTile"`
	CeilTile  int            `json:"ceilTile"`
	WallTiles map[string]int `json:"wallTiles"`
}

// LoadLevel reads and parses a level JSON file from disk.
func LoadLevel(path string) (*Level, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read level %q: %w", path, err)
	}
	var lvl Level
	if err := json.Unmarshal(data, &lvl); err != nil {
		return nil, fmt.Errorf("parse level %q: %w", path, err)
	}
	return &lvl, nil
}

// Camera returns a Camera initialised from the level's spawn point.
// The camera plane is derived perpendicular to the direction with a 0.66 FOV.
func (l *Level) Camera() *Camera {
	return &Camera{
		PosX:   l.Spawn.X,
		PosY:   l.Spawn.Y,
		DirX:   l.Spawn.DirX,
		DirY:   l.Spawn.DirY,
		PlaneX: -l.Spawn.DirY * 0.66,
		PlaneY: l.Spawn.DirX * 0.66,
	}
}

// parseGrid converts the string-based grid into a [][]int grid.
// Each character is treated as a hex digit: '0'=0, '1'=1, ..., 'F'/'f'=15.
func parseGrid(rows []string) [][]int {
	grid := make([][]int, len(rows))
	for r, line := range rows {
		row := make([]int, len(line))
		for c, ch := range line {
			switch {
			case ch >= '0' && ch <= '9':
				row[c] = int(ch - '0')
			case ch >= 'A' && ch <= 'F':
				row[c] = int(ch-'A') + 10
			case ch >= 'a' && ch <= 'f':
				row[c] = int(ch-'a') + 10
			}
		}
		grid[r] = row
	}
	return grid
}

// World returns a World initialised from the level's grid and tileset data.
func (l *Level) World() *World {
	grid := parseGrid(l.Grid)

	height := len(grid)
	width := 0
	if height > 0 {
		width = len(grid[0])
	}

	// Convert hex-keyed wall tile map ("1"–"F") into a 16-element lookup slice.
	wallTiles := make([]int, 16)
	for key, tileIdx := range l.Tileset.WallTiles {
		n, err := strconv.ParseInt(key, 16, 64)
		if err != nil || n < 0 || n > 15 {
			continue
		}
		wallTiles[n] = tileIdx
	}

	return &World{
		Grid:      grid,
		Width:     width,
		Height:    height,
		WallTiles: wallTiles,
		FloorTile: l.Tileset.FloorTile,
		CeilTile:  l.Tileset.CeilTile,
		TileSize:  l.Tileset.TileSize,
		TileCols:  l.Tileset.TileCols,
	}
}

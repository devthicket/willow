package raycast

import "math"

// Camera holds the player position, view direction, and camera plane (FOV).
type Camera struct {
	PosX, PosY     float64
	DirX, DirY     float64
	PlaneX, PlaneY float64
}

// Rotate rotates the camera direction and plane by the given angle (radians).
func (c *Camera) Rotate(angle float64) {
	cos, sin := math.Cos(angle), math.Sin(angle)
	c.DirX, c.DirY = c.DirX*cos-c.DirY*sin, c.DirX*sin+c.DirY*cos
	c.PlaneX, c.PlaneY = c.PlaneX*cos-c.PlaneY*sin, c.PlaneX*sin+c.PlaneY*cos
}

// TryMove attempts to move the camera by (mx, my) scaled by speed.
// Axes are tested independently so the player slides along walls.
// A corner check prevents clipping through diagonal wall junctions.
// The walkable callback returns true if the given grid cell is passable.
func (c *Camera) TryMove(mx, my, speed float64, walkable func(x, y int) bool) {
	const pad = 0.25

	dx := mx * speed
	dy := my * speed

	nx := c.PosX + dx
	padX := pad
	if dx < 0 {
		padX = -pad
	}
	if walkable(int(nx+padX), int(c.PosY)) {
		c.PosX = nx
	}

	ny := c.PosY + dy
	padY := pad
	if dy < 0 {
		padY = -pad
	}
	if walkable(int(c.PosX), int(ny+padY)) {
		c.PosY = ny
	}

	// Corner check: prevent clipping through diagonal wall junctions.
	if !walkable(int(c.PosX+padX), int(c.PosY+padY)) {
		c.PosX -= dx
		c.PosY -= dy
	}
}

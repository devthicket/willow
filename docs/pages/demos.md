# Demos

Live WASM demos running directly in your browser — no install required.

## 10k Sprites

<a href="demos/sprites10k/" target="_blank"><img src="demos/sprites10k/thumbnail.png" alt="10k Sprites demo screenshot" width="640"></a>

10,000 whelp sprites rotating, scaling, fading, and bouncing simultaneously. A stress test for the Willow rendering pipeline.

<a href="demos/sprites10k/" target="_blank">Launch Demo</a> · <a href="https://github.com/phanxgames/willow/tree/main/demos/sprites10k" target="_blank">Source Code</a>

## Physics Shapes

<a href="demos/physics/" target="_blank"><img src="demos/physics/thumbnail.png" alt="Physics Shapes demo screenshot" width="640"></a>

~50 random shapes (circles, squares, triangles, pentagons, hexagons) with gravity, collisions, and click-to-jump. Heavier shapes fall faster and are harder to push — click any shape to give it an upward impulse.

<a href="demos/physics/" target="_blank">Launch Demo</a> · <a href="https://github.com/phanxgames/willow/tree/main/demos/physics" target="_blank">Source Code</a>

## Rope Garden

<a href="demos/ropegarden/" target="_blank"><img src="demos/ropegarden/thumbnail.png" alt="Rope Garden demo screenshot" width="640"></a>

A cable-untangling puzzle. Eight color-coded cables connect sockets on the left to matching sockets on the right. The pegs start shuffled, creating a tangle — drag each peg to the socket that matches its color to straighten every cable and win.

<a href="demos/ropegarden/" target="_blank">Launch Demo</a> · <a href="https://github.com/phanxgames/willow/tree/main/demos/ropegarden" target="_blank">Source Code</a>

## Underwater

<a href="demos/underwater/" target="_blank"><img src="demos/underwater/thumbnail.png" alt="Underwater demo screenshot" width="640"></a>

A layered underwater scene revealed through a circular porthole mask that follows the cursor. Multiple animated water meshes at varying depths, swaying seaweed with rising bubbles, a swimming whelp, caustic lighting, and a sandy floor create an atmospheric underwater world. Click to spawn a bubble burst.

<a href="demos/underwater/" target="_blank">Launch Demo</a> · <a href="https://github.com/phanxgames/willow/tree/main/demos/underwater" target="_blank">Source Code</a>

---

Want to run demos locally? Clone the repo and use `go run`:

```bash
git clone https://github.com/phanxgames/willow.git
cd willow
go run ./demos/sprites10k
go run ./demos/physics
go run ./demos/ropegarden
go run ./demos/underwater
```

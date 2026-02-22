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

## Tween Gallery

<a href="demos/tweengallery/" target="_blank"><img src="demos/tweengallery/thumbnail.png" alt="Tween Gallery demo screenshot" width="640"></a>

An interactive showcase of Willow's tween animation system. Five whelp sprites each demonstrate a different tween property — position, scale, rotation, alpha, and color. Click any of the 12 easing function buttons to see the selected curve applied to all five properties simultaneously. Animations loop automatically for easy comparison.

<a href="demos/tweengallery/" target="_blank">Launch Demo</a> · <a href="https://github.com/phanxgames/willow/tree/main/demos/tweengallery" target="_blank">Source Code</a>

## Filter Gallery

<a href="demos/filtergallery/" target="_blank"><img src="demos/filtergallery/thumbnail.png" alt="Filter Gallery demo screenshot" width="640"></a>

An interactive showcase of every built-in Willow filter. A whelp sprite sits in the center with 10 toggle buttons below — click to activate filters individually or stack them together. Includes static filters (blur, sepia, outline, inline, pixel-perfect outline, grayscale, contrast) and animated ones (oscillating brightness, continuous hue rotation, palette cycling).

<a href="demos/filtergallery/" target="_blank">Launch Demo</a> · <a href="https://github.com/phanxgames/willow/tree/main/demos/filtergallery" target="_blank">Source Code</a>

## Lighting

<a href="demos/lighting/" target="_blank"><img src="demos/lighting/thumbnail.png" alt="Lighting demo screenshot" width="640"></a>

A dark dungeon scene showcasing the LightLayer system with heavy ambient darkness (96% opacity). Five colored torches flicker on pillars, three autonomous wisps drift through the scene, and a warm lantern follows the cursor. Click any torch to toggle it, or click empty space to spawn a temporary flash burst. Stone tiles, walls, crates, and gem clusters give the lighting something to reveal.

<a href="demos/lighting/" target="_blank">Launch Demo</a> · <a href="https://github.com/phanxgames/willow/tree/main/demos/lighting" target="_blank">Source Code</a>

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
go run ./demos/filtergallery
go run ./demos/tweengallery
go run ./demos/lighting
```

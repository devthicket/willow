# Examples & Demos

Willow ships with 20+ runnable examples covering everything from basic sprites to full scenes combining lighting, meshes, particles, and masks. Each one is a self-contained `main.go`.

```bash
git clone https://github.com/phanxgames/willow.git
cd willow
go run ./examples/<name>
```

Browse the [examples/ directory on GitHub](https://github.com/phanxgames/willow/tree/main/examples) for full source.

---

## Playable in Your Browser (WASM)

Try these directly  -  no install required. Click a thumbnail to launch.

<table>
<tr>
<td align="center" width="33%"><a id="10k-sprites"></a>
<a href="demos/sprites10k/" target="_blank"><img src="demos/sprites10k/thumbnail.png" alt="10k Sprites" width="240"></a><br>
<strong>10k Sprites</strong><br>
<a href="demos/sprites10k/" target="_blank">Launch</a> · <a href="https://github.com/phanxgames/willow/tree/main/examples/sprites10k" target="_blank">Source</a>
</td>
<td align="center" width="33%"><a id="physics-shapes"></a>
<a href="demos/physics/" target="_blank"><img src="demos/physics/thumbnail.png" alt="Physics Shapes" width="240"></a><br>
<strong>Physics Shapes</strong><br>
<a href="demos/physics/" target="_blank">Launch</a> · <a href="https://github.com/phanxgames/willow/tree/main/examples/physics" target="_blank">Source</a>
</td>
<td align="center" width="33%"><a id="rope-garden"></a>
<a href="demos/ropegarden/" target="_blank"><img src="demos/ropegarden/thumbnail.png" alt="Rope Garden" width="240"></a><br>
<strong>Rope Garden</strong><br>
<a href="demos/ropegarden/" target="_blank">Launch</a> · <a href="https://github.com/phanxgames/willow/tree/main/examples/ropegarden" target="_blank">Source</a>
</td>
</tr>
<tr>
<td align="center"><a id="tween-gallery"></a>
<a href="demos/tweengallery/" target="_blank"><img src="demos/tweengallery/thumbnail.png" alt="Tween Gallery" width="240"></a><br>
<strong>Tween Gallery</strong><br>
<a href="demos/tweengallery/" target="_blank">Launch</a> · <a href="https://github.com/phanxgames/willow/tree/main/examples/tweengallery" target="_blank">Source</a>
</td>
<td align="center"><a id="filter-gallery"></a>
<a href="demos/filtergallery/" target="_blank"><img src="demos/filtergallery/thumbnail.png" alt="Filter Gallery" width="240"></a><br>
<strong>Filter Gallery</strong><br>
<a href="demos/filtergallery/" target="_blank">Launch</a> · <a href="https://github.com/phanxgames/willow/tree/main/examples/filtergallery" target="_blank">Source</a>
</td>
<td align="center"><a id="lighting-demo"></a>
<a href="demos/lighting/" target="_blank"><img src="demos/lighting/thumbnail.png" alt="Lighting" width="240"></a><br>
<strong>Lighting</strong><br>
<a href="demos/lighting/" target="_blank">Launch</a> · <a href="https://github.com/phanxgames/willow/tree/main/examples/lighting" target="_blank">Source</a>
</td>
</tr>
<tr>
<td align="center"><a id="underwater"></a>
<a href="demos/underwater/" target="_blank"><img src="demos/underwater/thumbnail.png" alt="Underwater" width="240"></a><br>
<strong>Underwater</strong><br>
<a href="demos/underwater/" target="_blank">Launch</a> · <a href="https://github.com/phanxgames/willow/tree/main/examples/underwater" target="_blank">Source</a>
</td>
<td></td>
<td></td>
</tr>
</table>

---

## Basics

<table>
<tr>
<td><a id="basic"></a>
<strong>Basic</strong><br>
<code>go run ./examples/basic</code><br><br>
Bouncing colored sprite  -  the simplest possible Willow app. Creates a single solid-color sprite that moves and bounces off screen edges.
</td>
<td width="260"></td>
</tr>
<tr>
<td><a id="shapes"></a>
<strong>Shapes</strong><br>
<code>go run ./examples/shapes</code><br><br>
Rotating polygon hierarchy demonstrating nested transforms and solid-color sprites. Shows how child nodes inherit and compound parent transformations.
</td>
<td width="260"><img src="gif/shapes.gif" alt="Shapes" width="240"></td>
</tr>
<tr>
<td><a id="interaction"></a>
<strong>Interaction</strong><br>
<code>go run ./examples/interaction</code><br><br>
Draggable, clickable rectangles showcasing hit testing and gesture handling. Demonstrates <code>OnClick</code>, <code>OnDrag</code>, and pointer event routing through the scene graph.
</td>
<td width="260"></td>
</tr>
</table>

---

## Text

<table>
<tr>
<td><a id="spritefont"></a>
<strong>DistanceFieldFont</strong><br>
<code>go run ./examples/text</code><br><br>
DistanceFieldFont (SDF) text with alignment and wrapping options, plus effects (outline, glow, shadow). Generates an SDF atlas PNG at startup and saves it to the example directory.
</td>
<td width="260"></td>
</tr>
<tr>
<td><a id="spritefont-ttf"></a>
<strong>DistanceFieldFont (TTF)</strong><br>
<code>go run ./examples/texttf</code><br><br>
Runtime DistanceFieldFont generation from TTF data with outline support. Shows SDF effects (outline, glow, shadow) alongside TTF text. Saves the generated SDF atlas PNG.
</td>
<td width="260"></td>
</tr>
<tr>
<td><a id="pixelfont"></a>
<strong>PixelFont</strong><br>
<code>go run ./examples/text-pixelfont</code><br><br>
Pixel-perfect bitmap font from a spritesheet. Shows integer scaling (1x, 2x, 3x) via FontSize, color tinting, word wrapping, alignment, cell trimming, and the full character set.
</td>
<td width="260"></td>
</tr>
</table>

---

## Animation

<table>
<tr>
<td><a id="tweens"></a>
<strong>Tweens</strong><br>
<code>go run ./examples/tweens</code><br><br>
Position, scale, rotation, alpha, and color tweens using the built-in tween system. Click any sprite to restart its tween. Demonstrates all tween types and easing functions.
</td>
<td width="260"><img src="gif/tweens.gif" alt="Tweens" width="240"></td>
</tr>
<tr>
<td><a id="tween-gallery-ex"></a>
<strong>Tween Gallery</strong><br>
<code>go run ./examples/tweengallery</code><br><br>
Interactive showcase of every easing function. Five sprites each demonstrate a different tween property  -  position, scale, rotation, alpha, and color. Click easing buttons to compare curves.
</td>
<td width="260"></td>
</tr>
<tr>
<td><a id="particles"></a>
<strong>Particles</strong><br>
<code>go run ./examples/particles</code><br><br>
Fountain, campfire, and sparkler particle effects. Each emitter uses different configurations for speed, lifetime, color, and gravity to create distinct visual effects.
</td>
<td width="260"><img src="gif/particles.gif" alt="Particles" width="240"></td>
</tr>
</table>

---

## Visual Effects

<table>
<tr>
<td><a id="shaders"></a>
<strong>Shaders</strong><br>
<code>go run ./examples/shaders</code><br><br>
Built-in shader filters showcase  -  blur, glow, chromatic aberration, and more. Demonstrates applying Kage-based post-processing filters to nodes.
</td>
<td width="260"><img src="gif/shaders.gif" alt="Shaders" width="240"></td>
</tr>
<tr>
<td><a id="filter-gallery-ex"></a>
<strong>Filter Gallery</strong><br>
<code>go run ./examples/filtergallery</code><br><br>
Interactive showcase of every built-in filter. Toggle 10 filters on/off and stack them together  -  blur, sepia, outline, inline, pixel-perfect outline, grayscale, contrast, brightness, hue shift, and palette cycling.
</td>
<td width="260"></td>
</tr>
<tr>
<td><a id="outline"></a>
<strong>Outline</strong><br>
<code>go run ./examples/outline</code><br><br>
Outline and inline filters applied to sprites. Shows how to add colored outlines of varying thickness around any node.
</td>
<td width="260"></td>
</tr>
<tr>
<td><a id="masks"></a>
<strong>Masks</strong><br>
<code>go run ./examples/masks</code><br><br>
Star polygon, cursor-following, and erase masking techniques. Demonstrates <code>SetMask</code> and <code>ClearMask</code> for clipping rendered content to arbitrary shapes.
</td>
<td width="260"><img src="gif/masks.gif" alt="Masks" width="240"></td>
</tr>
<tr>
<td><a id="lighting"></a>
<strong>Lighting</strong><br>
<code>go run ./examples/lighting</code><br><br>
Dark dungeon scene with torches, wisps, flash bursts, and a cursor-following lantern. Showcases the LightLayer system with heavy ambient darkness.
</td>
<td width="260"><img src="gif/lights.gif" alt="Lighting" width="240"></td>
</tr>
</table>

---

## Sprites & Maps

<table>
<tr>
<td><a id="atlas"></a>
<strong>Atlas</strong><br>
<code>go run ./examples/atlas</code><br><br>
TexturePacker atlas loading and sprite sheet display. Loads a JSON atlas exported from TexturePacker and displays individual frames.
</td>
<td width="260"></td>
</tr>
<tr>
<td><a id="tilemap"></a>
<strong>Tilemap</strong><br>
<code>go run ./examples/tilemap</code><br><br>
Tile map rendering with camera panning. Builds a tile grid from a tileset image and moves the camera with arrow keys or dragging.
</td>
<td width="260"><img src="gif/tilemap.gif" alt="Tilemap" width="240"></td>
</tr>
<tr>
<td><a id="tilemap-viewport"></a>
<strong>Tilemap Viewport</strong><br>
<code>go run ./examples/tilemapviewport</code><br><br>
Tilemap rendered through a viewport with culling. Only visible tiles are drawn, demonstrating efficient large-map rendering.
</td>
<td width="260"></td>
</tr>
</table>

---

## Meshes

<table>
<tr>
<td><a id="rope"></a>
<strong>Rope</strong><br>
<code>go run ./examples/rope</code><br><br>
Draggable endpoints connected by a textured rope mesh. Demonstrates mesh-based rendering with per-vertex UV mapping and interactive control points.
</td>
<td width="260"></td>
</tr>
<tr>
<td><a id="water-mesh"></a>
<strong>Water Mesh</strong><br>
<code>go run ./examples/watermesh</code><br><br>
Water surface with per-vertex wave animation. Uses a mesh grid with vertices displaced by sine waves to create an animated water effect.
</td>
<td width="260"><img src="gif/watermesh.gif" alt="Water Mesh" width="240"></td>
</tr>
</table>

---

## Showcases

Larger examples that combine multiple Willow features into complete scenes.

<table>
<tr>
<td><a id="sprites10k"></a>
<strong>10k Sprites</strong><br>
<code>go run ./examples/sprites10k</code><br><br>
10,000 whelp sprites rotating, scaling, fading, and bouncing simultaneously. A stress test for the rendering pipeline.
</td>
<td width="260"></td>
</tr>
<tr>
<td><a id="physics"></a>
<strong>Physics Shapes</strong><br>
<code>go run ./examples/physics</code><br><br>
~50 random shapes with gravity, collisions, and click-to-jump. Heavier shapes fall faster and are harder to push.
</td>
<td width="260"></td>
</tr>
<tr>
<td><a id="ropegarden"></a>
<strong>Rope Garden</strong><br>
<code>go run ./examples/ropegarden</code><br><br>
A cable-untangling puzzle. Eight color-coded cables connect sockets  -  drag pegs to matching sockets to straighten every cable.
</td>
<td width="260"></td>
</tr>
<tr>
<td><a id="underwater-ex"></a>
<strong>Underwater</strong><br>
<code>go run ./examples/underwater</code><br><br>
Layered underwater scene revealed through a porthole mask. Combines water meshes, seaweed, particles, caustic lighting, and masking. Click to spawn bubble bursts.
</td>
<td width="260"></td>
</tr>
</table>

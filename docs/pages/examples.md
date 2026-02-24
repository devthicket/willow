# Examples

Willow ships with runnable examples covering everything from basic sprites to mesh distortion. Each one is a self-contained `main.go` you can run in a few seconds.

```bash
git clone https://github.com/phanxgames/willow.git
cd willow
go run ./examples/<name>
```

Browse the [examples/ directory on GitHub](https://github.com/phanxgames/willow/tree/main/examples) for full source.

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
<td><a id="text"></a>
<strong>Bitmap & SDF Font</strong><br>
<code>go run ./examples/text</code><br><br>
Bitmap font text with alignment and wrapping options, plus SDF text effects (outline, glow, shadow). Generates an SDF atlas PNG at startup and saves it to the example directory.
</td>
<td width="260"></td>
</tr>
<tr>
<td><a id="ttf-text"></a>
<strong>TTF & SDF Text</strong><br>
<code>go run ./examples/texttf</code><br><br>
TrueType font rendering with outline support, plus runtime SDF font generation from TTF. Shows SDF effects (outline, glow, shadow) alongside TTF text. Saves the generated SDF atlas PNG.
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
Dark dungeon scene with colored torch lights. Shows additive blending and light map compositing for dynamic 2D lighting.
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

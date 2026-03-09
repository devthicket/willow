package willow

import "github.com/phanxgames/willow/internal/atlas"

// AtlasManager is the global singleton that owns atlas page images.
// Pages are shared across Scenes, enabling fonts and atlases to be created
// independently of any particular Scene. Single-threaded (no sync needed).
type AtlasManager = atlas.Manager

// atlasManager returns the global AtlasManager singleton, lazily initialised.
func atlasManager() *AtlasManager {
	return atlas.GlobalManager()
}

// resetAtlasManager resets the global singleton for test isolation.
func resetAtlasManager() {
	atlas.ResetGlobalManager()
}

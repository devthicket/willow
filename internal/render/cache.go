package render

import (
	"github.com/devthicket/willow/internal/node"
	"github.com/devthicket/willow/internal/types"
)

// CacheTreeData holds subtree command cache state. Allocated on first
// SetCacheAsTree(true) call. Only a handful of nodes ever use this.
// Stored on Node.CacheData as an opaque `any`.
type CacheTreeData struct {
	Mode            types.CacheTreeMode
	Dirty           bool
	Commands        []CachedCmd
	ParentTransform [6]float32
	ParentAlpha     float32
}

// CachedCmd is a render command with transform/color stored at cache time.
// The two-tier source pointer supports animated tiles: nil = static (use
// cmd.TextureRegion), non-nil = animated (read live TextureRegion from node).
type CachedCmd struct {
	Cmd          RenderCommand // Transform/Color stored as-is at cache time
	Source       *node.Node    // nil = static, non-nil = animated (read live TextureRegion)
	SourceNodeID uint32        // emitting node's ID, for deferred animated registration lookup
}

// GetCacheTreeData extracts the CacheTreeData from a node's CacheData field.
// Returns nil if CacheData is nil or a different type.
func GetCacheTreeData(n *node.Node) *CacheTreeData {
	if n.CacheData == nil {
		return nil
	}
	ctd, _ := n.CacheData.(*CacheTreeData)
	return ctd
}

// SetCacheTreeData sets a node's CacheData to a CacheTreeData.
func SetCacheTreeData(n *node.Node, ctd *CacheTreeData) {
	n.CacheData = ctd
}

package render

import (
	"github.com/devthicket/willow/internal/node"
	"github.com/devthicket/willow/internal/types"
)

// SetCacheAsTree enables or disables subtree command caching on a node.
func SetCacheAsTree(n *node.Node, enabled bool, mode ...types.CacheTreeMode) {
	if enabled {
		if n.CacheData == nil {
			n.CacheData = &CacheTreeData{}
		}
		ct := GetCacheTreeData(n)
		if len(mode) > 0 {
			ct.Mode = mode[0]
		} else {
			ct.Mode = types.CacheTreeAuto
		}
		ct.Dirty = true
	} else {
		n.CacheData = nil
	}
}

// InvalidateCacheTree marks the node's cache as stale.
func InvalidateCacheTree(n *node.Node) {
	if n.CacheData != nil {
		GetCacheTreeData(n).Dirty = true
	}
}

// RegisterAnimatedInCache walks up to the nearest CacheAsTree ancestor and
// promotes this node's cachedCmd from static to animated so replay reads the
// live TextureRegion.
func RegisterAnimatedInCache(n *node.Node) {
	for p := n.Parent; p != nil; p = p.Parent {
		if p.CacheData == nil {
			continue
		}
		ct := GetCacheTreeData(p)
		if len(ct.Commands) == 0 {
			return
		}
		for i := range ct.Commands {
			if ct.Commands[i].Source == n || ct.Commands[i].SourceNodeID == n.ID {
				ct.Commands[i].Source = n
				return
			}
		}
		return
	}
}

// InvalidateAncestorCache walks up the tree from n to find the nearest
// CacheAsTree ancestor and marks it dirty (auto mode only).
func InvalidateAncestorCache(n *node.Node) {
	if n.CacheData != nil {
		ct := GetCacheTreeData(n)
		if ct.Mode == types.CacheTreeAuto {
			ct.Dirty = true
		}
	}
	for p := n.Parent; p != nil; p = p.Parent {
		if p.CacheData != nil {
			ct := GetCacheTreeData(p)
			if ct.Mode == types.CacheTreeAuto {
				ct.Dirty = true
			}
			return
		}
	}
}

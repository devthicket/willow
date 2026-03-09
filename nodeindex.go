package willow

import "github.com/phanxgames/willow/internal/node"

// NodeIndex is an opt-in registry for looking up nodes by name or tag.
// Data lives here, not on the Node struct, so there is zero overhead
// for nodes that are never indexed.
type NodeIndex = node.NodeIndex

// NewNodeIndex creates an empty NodeIndex.
var NewNodeIndex = node.NewNodeIndex

// wildMode describes the type of % wildcard match.
type wildMode = node.WildMode

// wildMode constants re-exported for test access.
const (
	wildNone     = node.WildNone
	wildPrefix   = node.WildPrefix
	wildSuffix   = node.WildSuffix
	wildContains = node.WildContains
)

// parsePattern checks for % wildcards at the start/end of pattern.
var parsePattern = node.ParsePattern

// matchPattern tests s against a parsed pattern.
var matchPattern = node.MatchPattern

package core

import "github.com/phanxgames/willow/internal/node"

// PropagateScene recursively sets the scene back-pointer on n and all descendants.
func PropagateScene(n *node.Node, s any) {
	if n.Scene_ == s {
		return
	}
	n.Scene_ = s
	for _, child := range n.Children_ {
		PropagateScene(child, s)
	}
}

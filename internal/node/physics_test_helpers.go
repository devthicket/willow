//go:build !nophysics

package node

import (
	"github.com/devthicket/willow/internal/physics"
	"github.com/devthicket/willow/internal/types"
)

// buildTree builds a flat tree of named container nodes from the given
// names; the first name is the root, the rest are direct children. Used
// by physics tests and items 10–13 to set up shared fixtures cheaply.
func buildTree(names ...string) (root *Node, byName map[string]*Node) {
	byName = make(map[string]*Node, len(names))
	for i, name := range names {
		n := NewNode(name, types.NodeTypeContainer)
		byName[name] = n
		if i == 0 {
			root = n
		} else {
			root.AddChild(n)
		}
	}
	return root, byName
}

// attachBody is a shorthand that calls SetBody with a Dynamic def of unit
// mass and a unit-radius circle shape. Tests that don't care about the
// specific shape use this to keep noise down.
func attachBody(n *Node, def physics.BodyDef) {
	if def == nil {
		def = physics.Dynamic{Shape: physics.Circle{Radius: 1}, Mass: 1}
	}
	n.SetBody(def)
}

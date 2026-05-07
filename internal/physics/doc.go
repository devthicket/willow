// Package physics is a thin wrapper over jakecoffman/cp/v2 (Chipmunk2D) that
// provides the type surface Willow's node layer composes against.
//
// This package is intentionally leaf: it imports cp and nothing inside Willow.
// Per-frame orchestration, tree-mutation bookkeeping, and the Node↔Body sync
// loop live in internal/node, which holds the only references back to *Node.
//
// Users do not import this package directly; the willow package re-exports
// the public surface as willow.PhysicsX aliases.
package physics

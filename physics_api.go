package willow

import "github.com/devthicket/willow/internal/physics"

// ---------------------------------------------------------------------------
// Physics (internal/physics)
// ---------------------------------------------------------------------------
//
// Physics types follow the Physics* prefix convention so they remain
// grep-friendly and unambiguous on the public surface. The verbs are methods
// on *Node (EnablePhysics, DisablePhysics, SetBody, RemoveBody, GetBody,
// StepPhysics), visible automatically through the Node = node.Node alias.

// PhysicsConfig configures a physics-enabled subtree.
type PhysicsConfig = physics.Config

// PhysicsBodyDef is the interface implemented by body definitions.
type PhysicsBodyDef = physics.BodyDef

// PhysicsDynamic describes a dynamic body driven by forces and impulses.
type PhysicsDynamic = physics.Dynamic

// PhysicsStatic describes an immovable body used for terrain and obstacles.
type PhysicsStatic = physics.Static

// PhysicsKinematic describes a body moved by user code, ignoring forces.
type PhysicsKinematic = physics.Kinematic

// PhysicsBody is the runtime body+shape pair attached to a Node.
type PhysicsBody = physics.Body

// PhysicsShape is the interface implemented by shape descriptors.
type PhysicsShape = physics.Shape

// PhysicsCircle is a circular shape descriptor.
type PhysicsCircle = physics.Circle

// PhysicsBox is an axis-aligned rectangular shape descriptor.
type PhysicsBox = physics.Box

// PhysicsSegment is a line-segment shape descriptor.
type PhysicsSegment = physics.Segment

// PhysicsPolygon is a convex polygon shape descriptor.
type PhysicsPolygon = physics.Polygon

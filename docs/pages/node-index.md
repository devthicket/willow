# Node Index

`NodeIndex` is an opt-in registry for looking up and grouping nodes by name or tag. All data lives in the index, not on nodes, so there is zero overhead for nodes that aren't indexed.

## Creating an Index

```go
idx := willow.NewNodeIndex()
```

## Registering Nodes

```go
// Register with tags
idx.Add(enemy, "enemy", "damageable")

// Register without tags (indexed by name only)
idx.Add(player)

// Add more tags later
idx.Add(enemy, "flying")
```

`Add` indexes the node by its `Name` field automatically. Calling `Add` again on the same node appends tags without duplicating the registration.

## Removing

```go
// Remove specific tags
idx.Remove(enemy, "flying")

// Remove node from index entirely
idx.Remove(enemy)
```

## Finding by Tag

```go
// Exact match
enemies := idx.FindByTag("enemy")

// All enemies that can take damage
targets := idx.FindByTags("enemy", "damageable")
```

`FindByTag` returns the internal slice directly for speed. Do not mutate the returned slice.

## Finding by Name

```go
// First match
boss := idx.FindByName("boss")

// All matches
allEnemies := idx.FindAllByName("enemy_grunt")
```

## Wildcard Patterns

All `Find` methods support `%` wildcards (SQL LIKE style):

| Pattern | Matches |
|---------|---------|
| `"enemy"` | Exact: `"enemy"` only |
| `"enemy%"` | Starts with: `"enemy_01"`, `"enemy_boss"` |
| `"%boss"` | Ends with: `"final_boss"`, `"mini_boss"` |
| `"%ene%"` | Contains: `"enemy"`, `"scene_manager"` |

Exact matches use O(1) map lookups. Wildcards scan keys but skip nodes that don't match.

```go
// Find all nodes whose name starts with "enemy_"
grunts := idx.FindAllByName("enemy%")

// Find all tags that start with "team_"
teamA := idx.FindByTag("team_%")
```

## Iteration

Iterate without allocating a result slice:

```go
// Iterate all nodes with a tag
idx.EachByTag("enemy", func(n *willow.Node) bool {
    n.SetAlpha(0.5)
    return true // return false to stop early
})

// Iterate all registered nodes
idx.Each(func(n *willow.Node) bool {
    fmt.Println(n.Name)
    return true
})

// Count without allocating
count := idx.CountByTag("enemy")
```

## Cookbook

### Damage all enemies in range

```go
idx.EachByTag("enemy", func(n *willow.Node) bool {
    dx := n.X() - blastX
    dy := n.Y() - blastY
    if dx*dx+dy*dy < radius*radius {
        // apply damage via UserData, ECS, or however you track HP
    }
    return true
})
```

### Despawn all bullets

```go
for _, b := range idx.FindByTag("bullet") {
    idx.Remove(b)
    b.Dispose()
}
```

### Tag-based collision groups

```go
// During spawn
idx.Add(playerBullet, "bullet", "player_bullet")
idx.Add(enemyBullet, "bullet", "enemy_bullet")

// Check only enemy bullets against player
for _, b := range idx.FindByTag("enemy_bullet") {
    if overlaps(player, b) {
        // hit
    }
}
```

### Find all nodes in a naming convention

```go
// Spawned as "enemy_001", "enemy_002", ...
allEnemies := idx.FindAllByName("enemy%")

// Find any node with "boss" somewhere in its name
bosses := idx.FindAllByName("%boss%")
```

### Multi-tag filtering

```go
// Only flying enemies that are also damageable
targets := idx.FindByTags("enemy", "flying", "damageable")
```

### Toggle a tag as state

```go
// Stun an enemy
idx.Add(enemy, "stunned")

// Unstun
idx.Remove(enemy, "stunned")

// Process stunned enemies
idx.EachByTag("stunned", func(n *willow.Node) bool {
    // skip AI, show stun effect, etc.
    return true
})
```

## Tree Search (Without Index)

For quick lookups without setting up an index, `Node` has built-in tree search with the same `%` wildcard support:

```go
// Direct children only
healthBar := enemyContainer.FindChild("health_bar")

// Recursive depth-first search
boss := scene.Root().FindDescendant("boss%")
```

These do O(n) tree walks per call. For repeated queries, use `NodeIndex` instead.

## Next Steps

- [Nodes](?page=nodes) - node types and tree manipulation
- [ECS Integration](?page=ecs-integration) - entity-component approach to node management

## Related

- [Events & Callbacks](?page=events-and-callbacks) - per-node interaction handlers
- [Performance](?page=performance-overview) - batching and allocation discipline

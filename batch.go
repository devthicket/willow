package willow

import (
	"github.com/phanxgames/willow/internal/render"
)

// batchKey groups render commands that can be submitted in a single draw call.
type batchKey = render.BatchKey

func commandBatchKey(cmd *RenderCommand) batchKey {
	return render.CommandBatchKey(cmd)
}

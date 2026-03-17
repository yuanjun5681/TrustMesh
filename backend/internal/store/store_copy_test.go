package store

import (
	"testing"

	"trustmesh/backend/internal/model"
)

func TestCopyTaskNormalizesNilSlices(t *testing.T) {
	task := &model.TaskDetail{
		Todos:     nil,
		Artifacts: nil,
		Result: model.TaskResult{
			Metadata: nil,
		},
	}

	cloned := copyTask(task)

	if cloned.Todos == nil {
		t.Fatal("expected todos to be normalized to an empty slice")
	}
	if cloned.Artifacts == nil {
		t.Fatal("expected artifacts to be normalized to an empty slice")
	}
	if cloned.Result.Metadata == nil {
		t.Fatal("expected result metadata to be normalized to an empty map")
	}
}

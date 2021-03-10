package knative

import (
	"testing"

	"github.com/iter8-tools/etc3/api/v2alpha2"
	"github.com/stretchr/testify/assert"
)

func TestMakeTask(t *testing.T) {
	task, err := MakeTask(&v2alpha2.TaskSpec{
		Task: "knative/init-experiment",
	})
	assert.NotEmpty(t, task)
	assert.NoError(t, err)

	task, err = MakeTask(&v2alpha2.TaskSpec{
		Task: "knative/init-experimental",
	})
	assert.Nil(t, task)
	assert.Error(t, err)
}

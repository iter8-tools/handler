package metrics

import (
	"encoding/json"
	"testing"

	"github.com/iter8-tools/etc3/api/v2alpha2"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func TestMakeFakeMetricsTask(t *testing.T) {
	_, err := MakeTask(&v2alpha2.TaskSpec{
		Task: LibraryName + "/" + "fake",
	})
	assert.Error(t, err)
}

func TestMakeTask(t *testing.T) {
	vers, err := json.Marshal([]Version{
		{
			Name: "test",
			URL:  "https://iter8.tools",
		},
	})
	task, err := MakeTask(&v2alpha2.TaskSpec{
		Task: "metrics/collect",
		With: map[string]v1.JSON{
			"versions": {Raw: vers},
		},
	})
	assert.NotEmpty(t, task)
	assert.NoError(t, err)

	task, err = MakeTask(&v2alpha2.TaskSpec{
		Task: "metrics/collect",
		With: map[string]v1.JSON{
			"versionables": {Raw: vers},
		},
	})
	assert.Empty(t, task)
	assert.Error(t, err)

	task, err = MakeTask(&v2alpha2.TaskSpec{
		Task: "metrics/collect-it",
	})
	assert.Nil(t, task)
	assert.Error(t, err)
}

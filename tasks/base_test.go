package tasks_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/iter8-tools/etc3/api/v2alpha2"
	"github.com/iter8-tools/handler/tasks"
	"github.com/iter8-tools/handler/tasks/lib/common"
	"github.com/stretchr/testify/assert"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func init() {
	log = tasks.GetLogger()
}

func TestWithoutExperiment(t *testing.T) {
	tags := tasks.GetDefaultTags(context.Background())
	assert.Empty(t, tags.M)
}

func TestWithExperiment(t *testing.T) {
	exp, err := (&tasks.Builder{}).FromFile(tasks.CompletePath("../", "testdata/experiment1.yaml")).Build()
	assert.NoError(t, err)
	ctx := context.WithValue(context.Background(), tasks.ContextKey("experiment"), exp)
	tags := tasks.GetDefaultTags(ctx)

	testStr := []string{
		"{{.this.apiVersion}}",
		"{{.this.metadata.name}}",
		"{{.this.spec.duration.intervalSeconds}}",
		"{{(index .this.spec.versionInfo.baseline.variables 0).value}}",
		"{{.this.status.versionRecommendedForPromotion}}",
	}
	expectedOut := []string{
		"iter8.tools/v2alpha2",
		"sklearn-iris-experiment-1",
		"15",
		"revision1",
		"default",
	}

	for i, in := range testStr {
		out, err := tags.Interpolate(&in)
		assert.NoError(t, err)
		assert.Equal(t, expectedOut[i], out)
	}
}

// multiple tasks successfully execute
func TestActionRun(t *testing.T) {
	action := tasks.Action{}
	script, _ := json.Marshal("echo hello")
	task, err := common.MakeTask(&v2alpha2.TaskSpec{
		Task: common.LibraryName + "/" + common.BashTaskName,
		With: map[string]apiextensionsv1.JSON{
			"script": {Raw: script},
		},
	})
	assert.NotEmpty(t, task)
	assert.NoError(t, err)
	action = append(action, task)

	script, _ = json.Marshal("echo goodbye")
	task, err = common.MakeTask(&v2alpha2.TaskSpec{
		Task: common.LibraryName + "/" + common.BashTaskName,
		With: map[string]apiextensionsv1.JSON{
			"script": {Raw: script},
		},
	})
	assert.NotEmpty(t, task)
	assert.NoError(t, err)
	action = append(action, task)

	exp, err := (&tasks.Builder{}).FromFile(tasks.CompletePath("../", "testdata/experiment10.yaml")).Build()
	assert.NoError(t, err)
	ctx := context.WithValue(context.Background(), tasks.ContextKey("experiment"), exp)

	a := &action
	err = a.Run(ctx)
	assert.NoError(t, err)
}

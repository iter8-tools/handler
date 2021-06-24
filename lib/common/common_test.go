package common

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/iter8-tools/etc3/api/v2alpha2"
	"github.com/iter8-tools/handler/base"
	"github.com/iter8-tools/handler/experiment"
	"github.com/iter8-tools/handler/utils"
	"github.com/stretchr/testify/assert"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func TestMakeTask(t *testing.T) {
	b, _ := json.Marshal("echo")
	a, _ := json.Marshal([]string{"hello", "people", "of", "earth"})
	task, err := MakeTask(&v2alpha2.TaskSpec{
		Task: "common/exec",
		With: map[string]apiextensionsv1.JSON{
			"cmd":  {Raw: b},
			"args": {Raw: a},
		},
	})
	assert.NotEmpty(t, task)
	assert.NoError(t, err)
	assert.Equal(t, "earth", task.(*ExecTask).With.Args[3])
	log.Trace(task.(*ExecTask).With.Args)

	exp, err := (&experiment.Builder{}).FromFile(utils.CompletePath("../../", "testdata/experiment10.yaml")).Build()
	task.Run(context.WithValue(context.Background(), base.ContextKey("experiment"), exp))

	task, err = MakeTask(&v2alpha2.TaskSpec{
		Task: "common/run",
		With: map[string]apiextensionsv1.JSON{
			"cmd": {Raw: b},
		},
	})
	assert.Nil(t, task)
	assert.Error(t, err)
}

func TestExecTaskNoInterpolation(t *testing.T) {
	b, _ := json.Marshal("echo")
	a, _ := json.Marshal([]string{"hello", "{{ omg }}", "world"})
	c, _ := json.Marshal(true)
	task, err := MakeTask(&v2alpha2.TaskSpec{
		Task: "common/exec",
		With: map[string]apiextensionsv1.JSON{
			"cmd":                  {Raw: b},
			"args":                 {Raw: a},
			"disableInterpolation": {Raw: c},
		},
	})

	assert.NotEmpty(t, task)
	assert.NoError(t, err)
	assert.Equal(t, "world", task.(*ExecTask).With.Args[2])
	log.Trace(task.(*ExecTask).With.Args)

	exp, err := (&experiment.Builder{}).FromFile(utils.CompletePath("../../", "testdata/experiment10.yaml")).Build()
	task.Run(context.WithValue(context.Background(), base.ContextKey("experiment"), exp))
}

func TestMakeBashTask(t *testing.T) {
	script, _ := json.Marshal("echo hello")
	task, err := MakeTask(&v2alpha2.TaskSpec{
		Task: LibraryName + "/" + BashTaskName,
		With: map[string]apiextensionsv1.JSON{
			"script": {Raw: script},
		},
	})
	assert.NotEmpty(t, task)
	assert.NoError(t, err)
	assert.Equal(t, "echo hello", task.(*BashTask).With.Script)
}

func TestBashRun(t *testing.T) {
	exp, err := (&experiment.Builder{}).FromFile(filepath.Join("..", "..", "testdata", "common", "bashexperiment.yaml")).Build()
	assert.NoError(t, err)
	actionSpec, err := exp.GetActionSpec("start")
	assert.NoError(t, err)
	// action, err := GetAction(exp, actionSpec)
	action, err := MakeTask(&actionSpec[0])
	assert.NoError(t, err)
	ctx := context.WithValue(context.Background(), base.ContextKey("experiment"), exp)
	err = action.Run(ctx)
	assert.NoError(t, err)
}

func TestMakePromoteKubectlTask(t *testing.T) {
	namespace, _ := json.Marshal("default")
	recursive, _ := json.Marshal(true)
	manifest, _ := json.Marshal("promote.yaml")
	task, err := MakeTask(&v2alpha2.TaskSpec{
		Task: LibraryName + "/" + PromoteKubectlTaskName,
		With: map[string]apiextensionsv1.JSON{
			"namespace": {Raw: namespace},
			"recursive": {Raw: recursive},
			"manifest":  {Raw: manifest},
		},
	})
	assert.NotEmpty(t, task)
	assert.NoError(t, err)
	assert.Equal(t, "default", *task.(*PromoteKubectlTask).With.Namespace)
	assert.Equal(t, "promote.yaml", task.(*PromoteKubectlTask).With.Manifest)
	assert.Equal(t, true, *task.(*PromoteKubectlTask).With.Recursive)

	bTask := *task.(*PromoteKubectlTask).ToBashTask()
	assert.Equal(t, "kubectl apply --namespace default --recursive --filename promote.yaml", bTask.With.Script)
}

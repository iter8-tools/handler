package common

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/iter8-tools/etc3/api/v2alpha2"
	"github.com/iter8-tools/handler/tasks"
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

	exp, err := (&tasks.Builder{}).FromFile(tasks.CompletePath("../../", "testdata/experiment10.yaml")).Build()
	task.Run(context.WithValue(context.Background(), tasks.ContextKey("experiment"), exp))

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

	exp, err := (&tasks.Builder{}).FromFile(tasks.CompletePath("../../", "testdata/experiment10.yaml")).Build()
	task.Run(context.WithValue(context.Background(), tasks.ContextKey("experiment"), exp))
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
	exp, err := (&tasks.Builder{}).FromFile(filepath.Join("..", "..", "..", "testdata", "common", "bashexperiment.yaml")).Build()
	assert.NoError(t, err)
	actionSpec, err := exp.GetActionSpec("start")
	assert.NoError(t, err)
	// action, err := GetAction(exp, actionSpec)
	action, err := MakeTask(&actionSpec[0])
	assert.NoError(t, err)
	ctx := context.WithValue(context.Background(), tasks.ContextKey("experiment"), exp)
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

func TestMakeReadinessTask(t *testing.T) {
	initDelay, _ := json.Marshal(5)
	numRetries, _ := json.Marshal(3)
	intervalSeconds, _ := json.Marshal(5)
	objRefs, _ := json.Marshal([]ObjRef{
		{
			Kind:      "deploy",
			Namespace: tasks.StringPointer("default"),
			Name:      "hello",
			WaitFor:   tasks.StringPointer("condition=available"),
		},
		{
			Kind:      "deploy",
			Namespace: tasks.StringPointer("default"),
			Name:      "hello-candidate",
			WaitFor:   tasks.StringPointer("condition=available"),
		},
	})
	task, err := MakeTask(&v2alpha2.TaskSpec{
		Task: LibraryName + "/" + ReadinessTaskName,
		With: map[string]apiextensionsv1.JSON{
			"initialDelaySeconds": {Raw: initDelay},
			"numRetries":          {Raw: numRetries},
			"intervalSeconds":     {Raw: intervalSeconds},
			"objRefs":             {Raw: objRefs},
		},
	})
	assert.NotEmpty(t, task)
	assert.NoError(t, err)
	assert.Equal(t, int32(5), *task.(*ReadinessTask).With.InitialDelaySeconds)
	assert.Equal(t, int32(3), *task.(*ReadinessTask).With.NumRetries)
	assert.Equal(t, int32(5), *task.(*ReadinessTask).With.IntervalSeconds)
	assert.Equal(t, 2, len(task.(*ReadinessTask).With.ObjRefs))
}

func TestMakeHttpRequestTask(t *testing.T) {
	url, _ := json.Marshal("http://postman-echo.com/post")
	body, _ := json.Marshal("{\"hello\":\"world\"}")
	headers, _ := json.Marshal([]v2alpha2.NamedValue{{
		Name:  "x-foo",
		Value: "bar",
	}, {
		Name:  "Authentication",
		Value: "Basic: dXNlcm5hbWU6cGFzc3dvcmQK",
	}})
	task, err := MakeTask(&v2alpha2.TaskSpec{
		Task: LibraryName + "/" + HTTPRequestTaskName,
		With: map[string]apiextensionsv1.JSON{
			"URL":     {Raw: url},
			"body":    {Raw: body},
			"headers": {Raw: headers},
		},
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, task)

	exp, err := (&tasks.Builder{}).FromFile(filepath.Join("..", "..", "testdata", "experiment1.yaml")).Build()
	assert.NoError(t, err)
	ctx := context.WithValue(context.Background(), tasks.ContextKey("experiment"), exp)

	req, err := task.(*HTTPRequestTask).prepareRequest(ctx)
	assert.NotEmpty(t, task)
	assert.NoError(t, err)

	assert.Equal(t, "http://postman-echo.com/post", req.URL.String())
	assert.Equal(t, "bar", req.Header.Get("x-foo"))

	err = task.(*HTTPRequestTask).Run(ctx)
	assert.NoError(t, err)
}

func TestTriggerAction(t *testing.T) {
	exp, err := (&tasks.Builder{}).FromFile(tasks.CompletePath("../../../", "testdata/trigger-experiment.yaml")).Build()
	assert.NoError(t, err)
	ctx := context.WithValue(context.Background(), tasks.ContextKey("experiment"), exp)

	taskSpec := exp.Experiment.Spec.Strategy.Actions["start"][0]
	task, err := MakeTask(&taskSpec)
	assert.NoError(t, err)
	assert.NotEmpty(t, task)

	req, err := task.(*HTTPRequestTask).prepareRequest(ctx)
	assert.NotEmpty(t, task)
	assert.NoError(t, err)

	dispatchURL := req.URL.String()
	runsURL := strings.Replace(req.URL.String(), "dispatches", "runs", -1)
	assert.Equal(t, "https://api.github.com/repos/kalantar/csvdiff/actions/workflows/hello.yaml/dispatches", dispatchURL)
	assert.Equal(t, "application/vnd.github.v3+json", req.Header.Get("Accept"))

	original := getNumRuns(t, runsURL)
	time.Sleep(10 * time.Second)

	err = task.(*HTTPRequestTask).Run(ctx)
	assert.NoError(t, err)
	time.Sleep(10 * time.Second)

	current := getNumRuns(t, runsURL)
	assert.Greater(t, current, original)
}

func getNumRuns(t *testing.T, runsURL string) int {
	resp, err := http.Get(runsURL)
	assert.NoError(t, err)
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	runs := make(map[string]interface{})
	err = json.Unmarshal(b, &runs)
	assert.NoError(t, err)

	x := runs["workflow_runs"].([]interface{})
	return len(x)
}

package common

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"testing"

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
	exp, err := (&tasks.Builder{}).FromFile(tasks.CompletePath("../../../", "testdata/experiment1.yaml")).Build()
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

func TestMakeHttpRequestTaskDefaults(t *testing.T) {
	url, _ := json.Marshal("http://target")
	task, err := MakeTask(&v2alpha2.TaskSpec{
		Task: LibraryName + "/" + HTTPRequestTaskName,
		With: map[string]apiextensionsv1.JSON{
			"URL": {Raw: url},
		},
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, task)

	exp, err := (&tasks.Builder{}).FromFile(tasks.CompletePath("../../../", "testdata/experiment1.yaml")).Build()
	assert.NoError(t, err)
	ctx := context.WithValue(context.Background(), tasks.ContextKey("experiment"), exp)

	req, err := task.(*HTTPRequestTask).prepareRequest(ctx)
	assert.NotEmpty(t, task)
	assert.NoError(t, err)

	assert.Equal(t, "POST", req.Method)
	assert.Equal(t, 1, len(req.Header))
	assert.Equal(t, "application/json", req.Header.Get("Content-type"))

	data, err := ioutil.ReadAll(req.Body)
	assert.NoError(t, err)

	expectedBody := `{"summary":{"winnerFound":false,"versionRecommendedForPromotion":"default"},"experiment":{"kind":"Experiment","apiVersion":"iter8.tools/v2alpha2","metadata":{"name":"sklearn-iris-experiment-1","namespace":"default","selfLink":"/apis/iter8.tools/v2alpha2/namespaces/default/experiments/sklearn-iris-experiment-1","uid":"b99489b6-a1b4-420f-9615-165d6ff88293","generation":2,"creationTimestamp":"2020-12-27T21:55:48Z","annotations":{"kubectl.kubernetes.io/last-applied-configuration":"{\"apiVersion\":\"iter8.tools/v2alpha2\",\"kind\":\"Experiment\",\"metadata\":{\"annotations\":{},\"name\":\"sklearn-iris-experiment-1\",\"namespace\":\"default\"},\"spec\":{\"criteria\":{\"indicators\":[\"95th-percentile-tail-latency\"],\"objectives\":[{\"metric\":\"mean-latency\",\"upperLimit\":1000},{\"metric\":\"error-rate\",\"upperLimit\":\"0.01\"}]},\"duration\":{\"intervalSeconds\":15,\"iterationsPerLoop\":10},\"strategy\":{\"type\":\"Canary\"},\"target\":\"default/sklearn-iris\"}}\n"}},"spec":{"target":"default/sklearn-iris","versionInfo":{"baseline":{"name":"default","variables":[{"name":"revision","value":"revision1"}]},"candidates":[{"name":"canary","variables":[{"name":"revision","value":"revision2"}],"weightObjRef":{"kind":"InferenceService","namespace":"default","name":"sklearn-iris","apiVersion":"serving.kubeflow.org/v1alpha2","fieldPath":".spec.canaryTrafficPercent"}}]},"strategy":{"testingPattern":"Canary","deploymentPattern":"Progressive","actions":{"finish":[{"task":"common/exec","with":{"args":["build","."],"cmd":"kustomize"}}],"start":[{"task":"common/exec","with":{"args":["hello-world","hello {{ revision }} world","hello {{ omg }} world"],"cmd":"echo"}},{"task":"common/exec","with":{"args":["v1","v2",20,40.5],"cmd":"helm"}}]},"weights":{"maxCandidateWeight":100,"maxCandidateWeightIncrement":10}},"criteria":{"requestCount":"request-count","indicators":["95th-percentile-tail-latency"],"objectives":[{"metric":"mean-latency","upperLimit":"1k"},{"metric":"error-rate","upperLimit":"10m"}],"strength":null},"duration":{"intervalSeconds":15,"iterationsPerLoop":10}},"status":{"conditions":[{"type":"Completed","status":"False","lastTransitionTime":"2020-12-27T21:55:49Z","reason":"StartHandlerLaunched","message":"Start handler 'start' launched"},{"type":"Failed","status":"False","lastTransitionTime":"2020-12-27T21:55:48Z"}],"initTime":"2020-12-27T21:55:48Z","lastUpdateTime":"2020-12-27T21:55:48Z","completedIterations":0,"versionRecommendedForPromotion":"default","message":"StartHandlerLaunched: Start handler 'start' launched"}}}`
	assert.Equal(t, expectedBody, string(data))
}

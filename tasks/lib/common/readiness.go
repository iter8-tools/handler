package common

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/iter8-tools/etc3/api/v2alpha2"
	"github.com/iter8-tools/handler/tasks"
)

const (
	// ReadinessTaskName is the name of the readiness task
	ReadinessTaskName string = "readiness"
)

// ObjRef contains details about a specific K8s object whose existence and readiness will be checked
type ObjRef struct {
	// Kind of the object. Specified in the TYPE[.VERSION][.GROUP] format used by `kubectl`
	// See https://kubernetes.io/docs/reference/generated/kubectl/kubectl-commands#get
	Kind string `json:"kind,omitempty" yaml:"kind,omitempty"`
	// Namespace of the object. Optional. If left unspecified, this will be defaulted to the namespace of the experiment
	Namespace *string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	// Name of the object
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
	// Wait for condition. Optional.
	// Any value that is accepted by the --for flag of the `kubectl wait` command can be specified.
	// See https://kubernetes.io/docs/reference/generated/kubectl/kubectl-commands#wait
	WaitFor *string `json:"waitFor,omitempty" yaml:"waitFor,omitempty"`
}

// ReadinessInputs contains a list of K8s object references along with
// optional readiness conditions for them. The inputs also specify the delays
// and retries involved in the existence and readiness checks.
type ReadinessInputs struct {
	// InitialDelaySeconds is optional and defaulted to 5 secs. The first check will be performed after this delay.
	InitialDelaySeconds *int32 `json:"initialDelaySeconds" yaml:"initialDelaySeconds"`
	// NumRetries is optional and defaulted to 12. This is the number of retries that will be attempted after the first check. Total number of trials = 1 + NumRetries.
	NumRetries *int32 `json:"numRetries" yaml:"numRetries"`
	// IntervalSeconds is optional and defaulted to 5 secs
	// Retries will be attempted periodically every IntervalSeconds
	IntervalSeconds *int32 `json:"intervalSeconds" yaml:"intervalSeconds"`
	// ObjRefs is a list of K8s objects along with optional readiness conditions
	ObjRefs []ObjRef `json:"objRefs" yaml:"objRefs"`
}

// ReadinessTask checks existence and readiness of specified resources
type ReadinessTask struct {
	tasks.TaskMeta `json:",inline" yaml:",inline"`
	With           ReadinessInputs `json:"with" yaml:"with"`
}

// MakeReadinessTask creates a readiness task with correct defaults.
func MakeReadinessTask(t *v2alpha2.TaskSpec) (tasks.Task, error) {
	if t.Task != LibraryName+"/"+ReadinessTaskName {
		return nil, fmt.Errorf("library and task need to be '%s' and '%s'", LibraryName, ReadinessTaskName)
	}
	var jsonBytes []byte
	// convert t to jsonBytes
	jsonBytes, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}
	// convert jsonString to ReadinessTask
	task := &ReadinessTask{}
	err = json.Unmarshal(jsonBytes, &task)
	if err != nil {
		return nil, err
	}
	// set defaults
	if task.With.InitialDelaySeconds == nil {
		task.With.InitialDelaySeconds = tasks.Int32Pointer(5)
	}
	if task.With.NumRetries == nil {
		task.With.NumRetries = tasks.Int32Pointer(12)
	}
	if task.With.IntervalSeconds == nil {
		task.With.IntervalSeconds = tasks.Int32Pointer(5)
	}
	return task, err
}

// Check existence and readiness of K8s objects.
func (t *ReadinessTask) Run(ctx context.Context) error {
	exp, err := tasks.GetExperimentFromContext(ctx)
	if err != nil {
		log.Error(err)
		return err
	}
	log.Trace("experiment", exp)

	time.Sleep(time.Duration(*t.With.InitialDelaySeconds) * time.Second)
	// invariant: objIndex is the number of objects that have been checked and found to be good
	objIndex := 0
	for i := 0; i <= int(*t.With.NumRetries); i++ {
		// this inner loop has no busy waiting (sleeps)
		// it will keep going through the objects as much as possible
		for err == nil && objIndex < len(t.With.ObjRefs) {
			// fix namespace
			var namespace string
			if t.With.ObjRefs[i].Namespace == nil {
				namespace = exp.Namespace
			} else {
				namespace = *t.With.ObjRefs[i].Namespace
			}
			// check existence
			script := fmt.Sprintf("kubectl get %s %s -n %s", t.With.ObjRefs[i].Kind, t.With.ObjRefs[i].Name, namespace)
			cmd := exec.Command("/bin/bash", "-c", script)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			log.Info("Executing command: " + cmd.String())
			err = cmd.Run()
			if err == nil {
				// check readiness condition if any
				if t.With.ObjRefs[i].WaitFor != nil {
					script := fmt.Sprintf("kubectl wait %s/%s -n %s --for=%s --timeout=0s", t.With.ObjRefs[i].Kind, t.With.ObjRefs[i].Name, namespace, *t.With.ObjRefs[i].WaitFor)
					cmd := exec.Command("/bin/bash", "-c", script)
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					log.Info("Executing command: " + cmd.String())
					err = cmd.Run()
				}
			}

			if err == nil {
				// advance objIndex
				objIndex++
			}
		}

		if i == int(*t.With.NumRetries) || objIndex == len(t.With.ObjRefs) { // we are done
			break // out of the for loop
		} else {
			// try again later
			time.Sleep(time.Duration(*t.With.IntervalSeconds) * time.Second)
		}
	}

	return err
}

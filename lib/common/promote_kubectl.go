package common

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/iter8-tools/etc3/api/v2alpha2"
	"github.com/iter8-tools/handler/base"
	"github.com/spf13/viper"
)

const (
	// PromoteKubectlTaskName is the name of the task
	PromoteKubectlTaskName string = "promote-kubectl"
)

// PromoteKubectlInputs contain the name and arguments of the command to be executed.
type PromoteKubectlInputs struct {
	Manifest string `json:"manifest" yaml:"manifest"`
	//+optional
	Namespace *string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	//+optional
	Recursive *bool `json:"recursive,omitempty" yaml:"recursive,omitempty"`
}

// PromoteKubectlTask encapsulates a promotion details.
type PromoteKubectlTask struct {
	base.TaskMeta `json:",inline" yaml:",inline"`
	With          PromoteKubectlInputs `json:"with" yaml:"with"`
}

// MakePromoteKubectlTask converts a task spec into an task.
func MakePromoteKubectlTask(t *v2alpha2.TaskSpec) (base.Task, error) {
	if t.Task != LibraryName+"/"+PromoteKubectlTaskName {
		return nil, fmt.Errorf("library and task need to be '%s' and '%s'", LibraryName, PromoteKubectlTaskName)
	}
	var jsonBytes []byte
	var task base.Task
	// convert t to jsonBytes
	jsonBytes, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}
	// convert jsonString to PromoteKubectlTask
	task = &PromoteKubectlTask{}
	err = json.Unmarshal(jsonBytes, &task)
	return task, err
}

// ToBashTask converts a PromoteKubectl task to a Bash task
func (t *PromoteKubectlTask) ToBashTask() *BashTask {
	namespace := t.With.Namespace
	if namespace == nil {
		ns := viper.GetViper().GetString("experiment_namespace")
		namespace = &ns
	}

	script := "kubectl apply --namespace " + *namespace
	if t.With.Recursive != nil && *t.With.Recursive {
		script += " --recursive"
	}
	script += " --filename " + t.With.Manifest

	tSpec := &BashTask{
		TaskMeta: base.TaskMeta{
			Library: LibraryName,
			Task:    BashTaskName,
		},
		With: BashInputs{
			Script: script,
		},
	}
	return tSpec
}

// Run the command.
func (t *PromoteKubectlTask) Run(ctx context.Context) error {
	return t.ToBashTask().Run(ctx)
}

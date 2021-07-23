package gitops

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/iter8-tools/etc3/api/v2alpha2"
	"github.com/iter8-tools/handler/tasks"
)

/*
- task: gitops/helmex-update
  with:
    expTemplate: experiment.yaml # optional with default value experiment.yaml; the experiment yaml in the helm chart
*/

const (
	// HelmexUpdateTaskName is the name of the task this file implements
	HelmexUpdateTaskName string = "helmex-update"

	// DefaultExpTemplate is the default value of the name of experiment template file in the Helmex chart
	DefaultExpTemplate string = "experiment.yaml"

	// DefaultPullRequest is the default value of the pullRequest field in this task. True. By default, this task will create a PR and not commit.
	DefaultPullRequest bool = true
)

// HelmexUpdateInputs contain the inputs to the helmex-update task to be executed.
type HelmexUpdateInputs struct {
	// GitRepo is the git repo
	GitRepo string `json:"gitRepo" yaml:"gitRepo"`
	// FilePath is the path to values.yaml file within the repo
	FilePath string `json:"filePath" yaml:"filePath"`
	// PullRequest indicates if this task will issue a PR or push directly
	PullRequest *bool `json:"pullRequest,omitempty" yaml:"pullRequest,omitempty"`
	// HelmRepo is the Helm repo used in the Helmex
	HelmRepo string `json:"helmRepo" yaml:"helmRepo"`
	// Chart is the Helm chart used in the Helmex
	Chart string `json:"chart" yaml:"chart"`
	// ExpTemplate is the name of the experiment template file
	ExpTemplate *string `json:"expTemplate,omitempty" yaml:"expTemplate,omitempty"`
}

// HelmexUpdateTask enables updates to the values.yaml file within a Helmex git repo.
type HelmexUpdateTask struct {
	tasks.TaskMeta
	With HelmexUpdateInputs `json:"with" yaml:"with"`
}

// MakeHelmexUpdate constructs a HelmexUpdateTask out of a task spec
func MakeHelmexUpdate(t *v2alpha2.TaskSpec) (tasks.Task, error) {
	if t.Task != LibraryName+"/"+HelmexUpdateTaskName {
		return nil, errors.New("library and task need to be " + LibraryName + " and " + HelmexUpdateTaskName)
	}
	var err error
	var jsonBytes []byte
	var bt tasks.Task
	// convert t to jsonBytes
	jsonBytes, err = json.Marshal(t)
	// convert jsonString to CollectTask
	if err == nil {
		hut := &HelmexUpdateTask{}
		err = json.Unmarshal(jsonBytes, &hut)
		bt = hut
	}
	return bt, err
}

// InitializeDefaults sets default values for HelmexUpdateTaskInputs
func (t *HelmexUpdateTask) InitializeDefaults() {
	if t.With.PullRequest == nil {
		t.With.PullRequest = tasks.BoolPointer(true)
	}
	if t.With.ExpTemplate == nil {
		t.With.ExpTemplate = tasks.StringPointer("experiment.yaml")
	}
}

// CloneGitRepo locally
func (t *HelmexUpdateTask) CloneGitRepo() error {
	return nil
}

// VerifyExperimentHash ensures that the current experiment is the same one that was created using the values file in the git repo.
// If the candidate section of the values file was updated since the creation of this experiment, this function will return an error.
// For this to work, experiment name needs to have a suffic computed from the candidate object in the values file.
// This is a requirement in Helmex experiments.
func (t *HelmexUpdateTask) VerifyExperimentHash() error {
	return nil
}

// UpdateValuesFile updates the locally cloned values.yaml file.
// ToDo: implement weight updates.
func (t *HelmexUpdateTask) UpdateValuesFile() error {
	return nil
}

// UpdateInGit updates values.yaml file in git by submitting a PR.
// ToDo: implement git push
func (t *HelmexUpdateTask) UpdateInGit() error {
	return nil
}

// Run executes the gitops/helmex-update task
func (t *HelmexUpdateTask) Run(ctx context.Context) error {
	log.Trace("collect task run started...")
	t.InitializeDefaults()
	err := t.CloneGitRepo()
	if err != nil {
		return err
	}
	err = t.VerifyExperimentHash()
	if err != nil {
		return err
	}
	err = t.UpdateValuesFile()
	if err != nil {
		return err
	}
	err = t.UpdateInGit()
	return err
}

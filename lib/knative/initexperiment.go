package knative

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/iter8-tools/etc3/api/v2alpha2"
	"github.com/iter8-tools/handler/base"
	"github.com/iter8-tools/handler/experiment"
	"k8s.io/apimachinery/pkg/types"

	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

// InitExperimentTask enables initialization of knative experiments.
type InitExperimentTask struct {
	Library string `json:"library" yaml:"library"`
	Task    string `json:"task" yaml:"task"`
}

// Run executes an InitExperimentTask
func (t *InitExperimentTask) Run(ctx context.Context) error {
	log.Trace("init experiment task run started...")
	var e *experiment.Experiment
	var err error
	if e, err = experiment.GetExperimentFromContext(ctx); err == nil {
		var nn *types.NamespacedName
		if nn, err = GetNamespacedNameForKsvc(e); err == nil {
			var ksvc *servingv1.Service
			log.Trace("Getting svc with namespaced name... ", *nn)
			if ksvc, err = GetKnativeSvc(nn); err == nil {
				if err = checkKsvcReadiness(ksvc); err == nil {
					if err = updateLocalExp(e, ksvc); err == nil {
						err = experiment.UpdateInClusterExperiment(e)
					}
				}
			}
		}
	}
	return err
}

// MakeInitExperiment converts an InitExperiment task spec into an InitExperimentTask.
func MakeInitExperiment(t *v2alpha2.TaskSpec) (base.Task, error) {
	if t.Task != "knative/init-experiment" {
		return nil, errors.New("library and task need to be 'knative' and 'init-experiment'")
	}
	var err error
	var jsonBytes []byte
	var it base.Task
	// convert t to jsonBytes
	jsonBytes, err = json.Marshal(t)
	// convert jsonString to InitExperimentTask
	if err == nil {
		it = &InitExperimentTask{}
		err = json.Unmarshal(jsonBytes, &it)
	}
	return it, err
}

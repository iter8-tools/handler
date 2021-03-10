package knative

import (
	"errors"

	"github.com/iter8-tools/etc3/api/v2alpha2"
	"github.com/iter8-tools/handler/base"
	"github.com/iter8-tools/handler/utils"
	"github.com/sirupsen/logrus"
)

var log *logrus.Logger

func init() {
	log = utils.GetLogger()
}

// MakeTask constructs a Task from a TaskMeta or returns an error if any.
func MakeTask(t *v2alpha2.TaskSpec) (base.Task, error) {
	switch t.Task {
	case "knative/init-experiment":
		return MakeInitExperiment(t)
	default:
		return nil, errors.New("Unknown task: " + t.Task)
	}
}

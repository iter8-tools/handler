package metrics

import (
	"errors"

	"github.com/iter8-tools/etc3/api/v2alpha2"
	"github.com/iter8-tools/handler/base"
	"github.com/iter8-tools/handler/utils"
	"github.com/sirupsen/logrus"
)

// Declare logger only once per package (in any file belonging to that package)
var log *logrus.Logger

func init() {
	// always use logger from utils
	// init logger once per package (in any file belonging to that package)
	log = utils.GetLogger()
}

// MakeTask constructs a Task from a TaskMeta or returns an error if any.
func MakeTask(t *v2alpha2.TaskSpec) (base.Task, error) {
	switch t.Task {
	case "metrics/collect":
		return MakeCollect(t)
	default:
		return nil, errors.New("Unknown task: " + t.Task)
	}
}

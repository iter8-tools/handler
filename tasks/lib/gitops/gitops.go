package gitops

import (
	"errors"

	"github.com/iter8-tools/etc3/api/v2alpha2"
	"github.com/iter8-tools/handler/tasks"
	"github.com/sirupsen/logrus"
)

const (
	// LibraryName is the name of this task library
	LibraryName string = "gitops"
)

var log *logrus.Logger

func init() {
	log = tasks.GetLogger()
}

// MakeTask constructs a Task from a TaskMeta or returns an error if any.
func MakeTask(t *v2alpha2.TaskSpec) (tasks.Task, error) {
	switch t.Task {
	case LibraryName + "/" + HelmexUpdateTaskName:
		hut, err := MakeHelmexUpdate(t)
		return hut, err
	default:
		return nil, errors.New("Unknown task: " + t.Task)
	}
}

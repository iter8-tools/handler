package notification

import (
	"errors"

	"github.com/iter8-tools/etc3/api/v2alpha2"
	"github.com/iter8-tools/handler/tasks"
	"github.com/sirupsen/logrus"
)

const (

	// LibraryName is the name of the task library this package implements
	LibraryName string = "notification"
)

var log *logrus.Logger

func init() {
	log = tasks.GetLogger()
}

// Inputs contains inputs common to all notifations.
// Should be included in the indivodual input structs via inline
type Inputs struct {
	IgnoreFailure *bool `json:"ignoreFailure,omitempty" yaml:"ignoreFailure,omitempty"`
}

// MakeTask constructs a Task from a TaskMeta or returns an error if any.
func MakeTask(t *v2alpha2.TaskSpec) (tasks.Task, error) {
	switch t.Task {
	case LibraryName + "/" + SlackTaskName:
		return MakeSlackTask(t)
	case LibraryName + "/" + HTTPTaskName:
		return MakeHTTPTask(t)
	// add additional tasks here
	default:
		return nil, errors.New("Unknown task: " + t.Task)
	}
}

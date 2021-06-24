package common

import (
	"errors"

	"github.com/iter8-tools/etc3/api/v2alpha2"
	"github.com/iter8-tools/handler/tasks"
	"github.com/sirupsen/logrus"
)

const (
	// LibraryName is the name of this task library
	LibraryName string = "common"
)

var log *logrus.Logger

func init() {
	log = tasks.GetLogger()
}

// MakeTask constructs a Task from a TaskSpec or returns an error if any.
func MakeTask(t *v2alpha2.TaskSpec) (tasks.Task, error) {
	switch t.Task {
	case LibraryName + "/" + ExecTaskName:
		return MakeExec(t)
	case LibraryName + "/" + BashTaskName:
		return MakeBashTask(t)
	case LibraryName + "/" + PromoteKubectlTaskName:
		return MakePromoteKubectlTask(t)
	case LibraryName + "/" + ReadinessTaskName:
		return MakeReadinessTask(t)
	case LibraryName + "/" + HttpRequestTaskName:
		return MakeHttpRequestTask(t)
	default:
		return nil, errors.New("Unknown task: " + t.Task)
	}
}

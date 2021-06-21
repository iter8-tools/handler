package common

import (
	"errors"

	"github.com/iter8-tools/etc3/api/v2alpha2"
	"github.com/iter8-tools/handler/base"
	"github.com/iter8-tools/handler/utils"
	"github.com/sirupsen/logrus"
)

const (
	// LibraryName is the name of this task library
	LibraryName string = "common"
)

var log *logrus.Logger

func init() {
	log = utils.GetLogger()
}

// MakeTask constructs a Task from a TaskSpec or returns an error if any.
func MakeTask(t *v2alpha2.TaskSpec) (base.Task, error) {
	switch t.Task {
	case LibraryName + "/" + ExecTaskName:
		return MakeExec(t)
	case LibraryName + "/" + BashTaskName:
		return MakeBashTask(t)
	case LibraryName + "/" + PromoteKubectlTaskName:
		return MakePromoteKubectlTask(t)
	default:
		return nil, errors.New("Unknown task: " + t.Task)
	}
}

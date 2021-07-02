package tasks

import (
	"context"
)

func init() {
	log = GetLogger()
}

// Task defines common method signatures for every task.
type Task interface {
	Run(ctx context.Context) error
}

// Action is a slice of Tasks.
type Action []Task

// TaskMeta is common to all Tasks
type TaskMeta struct {
	Library string `json:"library" yaml:"library"`
	Task    string `json:"task" yaml:"task"`
}

// Run the given action.
func (a *Action) Run(ctx context.Context) error {
	for i := 0; i < len(*a); i++ {
		log.Info("------")
		err := (*a)[i].Run(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetDefaultTags creates interpolation.Tags from experiment referenced by context
func GetDefaultTags(ctx context.Context) *Tags {
	tags := NewTags()
	exp, err := GetExperimentFromContext(ctx)
	if err == nil {
		obj, err := exp.ToMap()
		if err == nil {
			tags = tags.
				With("this", obj).
				WithRecommendedVersionForPromotion(&exp.Experiment)
		}
	} else {
		log.Warn("No experiment found in context")
	}

	return &tags
}

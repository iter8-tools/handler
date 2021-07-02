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

<<<<<<< HEAD
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
=======
// Tags supports string extrapolation using tags.
type Tags struct {
	M map[string]interface{}
}

// NewTags creates an empty instance of Tags
func NewTags() Tags {
	return Tags{M: make(map[string]interface{})}
}

// WithSecret adds the fields in secret to tags
func (tags Tags) WithSecret(key string, secret *corev1.Secret) Tags {
	obj := make(map[string]interface{})
	if secret != nil {
		for n, v := range secret.Data {
			obj[n] = string(v)
			// tags.M[n] = string(v)
		}
	}
	tags.M[key] = obj
	return tags
}

// With adds obj to tags
func (tags Tags) With(label string, obj interface{}) Tags {
	if obj != nil {
		tags.M[label] = obj
	}
	return tags
}

// WithRecommendedVersionForPromotion adds variables from versionDetail of version recommended for promotion
func (tags Tags) WithRecommendedVersionForPromotion(exp *v2alpha2.Experiment) Tags {
	if exp == nil || exp.Status.VersionRecommendedForPromotion == nil {
		log.Warn("no version recommended for promotion")
		return tags
	}

	versionRecommendedForPromotion := *exp.Status.VersionRecommendedForPromotion
	if exp.Spec.VersionInfo == nil {
		log.Warnf("No version details found for version recommended for promotion: %s", versionRecommendedForPromotion)
		return tags
	}

	var versionDetail *v2alpha2.VersionDetail = nil
	if exp.Spec.VersionInfo.Baseline.Name == versionRecommendedForPromotion {
		versionDetail = &exp.Spec.VersionInfo.Baseline
>>>>>>> 9a576e0 (modify secret support for tasks)
	} else {
		log.Warn("No experiment found in context")
	}

	return &tags
}

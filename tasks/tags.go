package tasks

import (
	"bytes"
	"errors"
	"html/template"

	"github.com/iter8-tools/etc3/api/v2alpha2"
	corev1 "k8s.io/api/core/v1"
)

// Tags supports string extrapolation using tags.
type Tags struct {
	M map[string]interface{}
}

// NewTags creates an empty instance of Tags
func NewTags() Tags {
	return Tags{M: make(map[string]interface{})}
}

// WithSecret adds the fields in secret to tags
func (tags Tags) WithSecret(label string, secret *corev1.Secret) Tags {
	if secret != nil {
		obj := make(map[string]interface{})
		for n, v := range secret.Data {
			obj[n] = string(v)
		}
		tags = tags.With(label, obj)
	}
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
	} else {
		for _, v := range exp.Spec.VersionInfo.Candidates {
			if v.Name == versionRecommendedForPromotion {
				versionDetail = &v
				break
			}
		}
	}
	if versionDetail == nil {
		log.Warnf("No version details found for version recommended for promotion: %s", versionRecommendedForPromotion)
		return tags
	}

	// get the variable values from the (recommended) versionDetail
	tags.M["name"] = versionDetail.Name
	for _, v := range versionDetail.Variables {
		tags.M[v.Name] = v.Value
	}

	return tags
}

// Interpolate str using tags.
func (tags *Tags) Interpolate(str *string) (string, error) {
	if tags == nil || tags.M == nil { // return a copy of the string
		return *str, nil
	}
	var err error
	var templ *template.Template
	if templ, err = template.New("").Parse(*str); err == nil {
		buf := bytes.Buffer{}
		if err = templ.Execute(&buf, tags.M); err == nil {
			return string(buf.Bytes()), nil
		}
		log.Error("template execution error: ", err)
		return "", errors.New("cannot interpolate string")
	}
	log.Error("template creation error: ", err)
	return "", errors.New("cannot interpolate string")
}

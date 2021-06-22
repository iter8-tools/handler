package base

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/iter8-tools/etc3/api/v2alpha2"
	"github.com/iter8-tools/handler/utils"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	log = utils.GetLogger()
}

func TestInterpolate(t *testing.T) {
	tags := NewTags().
		With("name", "tester").
		With("revision", "revision1").
		With("container", "super-container")

	// success cases
	inputs := []string{
		// `hello {{index . "name"}}`,
		// "hello {{index .name}}",
		"hello {{.name}}",
		"hello {{.name}}{{.other}}",
	}
	for _, str := range inputs {
		interpolated, err := tags.Interpolate(&str)
		assert.NoError(t, err)
		assert.Equal(t, "hello tester", interpolated)
	}

	// failure cases
	inputs = []string{
		// bad braces,
		"hello {{{index .name}}",
		// missing '.'
		"hello {{name}}",
	}
	for _, str := range inputs {
		_, err := tags.Interpolate(&str)
		assert.Error(t, err)
	}

	// empty tags (success cases)
	str := "hello {{.name}}"
	tags = NewTags()
	interpolated, err := tags.Interpolate(&str)
	assert.NoError(t, err)
	assert.Equal(t, "hello ", interpolated)

	// secret
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"secretName": []byte("tester"),
		},
	}

	str = "hello {{.secretName}}"
	tags = NewTags().WithSecret(&secret)
	assert.Contains(t, tags.M, "secretName")
	interpolated, err = tags.Interpolate(&str)
	assert.NoError(t, err)
	assert.Equal(t, "hello tester", interpolated)
}

func TestWithVersionRecommendedForPromotion(t *testing.T) {
	var data []byte
	data, err := ioutil.ReadFile(filepath.Join("..", "testdata", "experiment1.yaml"))
	assert.NoError(t, err)
	exp := &v2alpha2.Experiment{}
	err = yaml.Unmarshal(data, exp)
	assert.NoError(t, err)
	tags := NewTags().WithRecommendedVersionForPromotion(exp)
	assert.Equal(t, "revision1", tags.M["revision"])
}

func TestWithOutVersionRecommendedForPromotion(t *testing.T) {
	var data []byte
	data, err := ioutil.ReadFile(filepath.Join("..", "testdata", "experiment1-norecommended.yaml"))
	assert.NoError(t, err)
	exp := &v2alpha2.Experiment{}
	err = yaml.Unmarshal(data, exp)
	assert.NoError(t, err)
	tags := NewTags().WithRecommendedVersionForPromotion(exp)
	assert.NotContains(t, tags.M, "revision1")
	// assert.Equal(t, "revision1", tags.M["revision"])
}

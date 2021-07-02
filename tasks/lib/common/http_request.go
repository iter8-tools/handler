package common

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/iter8-tools/etc3/api/v2alpha2"
	"github.com/iter8-tools/handler/tasks"
)

const (
	// HTTPRequestTaskName is the name of the HTTP request task
	HTTPRequestTaskName string = "http-request"
)

// HTTPRequestInputs contain the name and arguments of the task.
type HTTPRequestInputs struct {
	URL      string                `json:"URL" yaml:"URL"`
	Method   *v2alpha2.MethodType  `json:"method,omitempty" yaml:"method,omitempty"`
	AuthType *v2alpha2.AuthType    `json:"authType,omitempty" yaml:"authType,omitempty"`
	Secret   *string               `json:"secret,omitempty" yaml:"secret,omitempty"`
	Headers  []v2alpha2.NamedValue `json:"headers,omitempty" yaml:"headers,omitempty"`
	Body     *string               `json:"body,omitempty" yaml:"body,omitempty"`
}

// HTTPRequestTask encapsulates the task.
type HTTPRequestTask struct {
	tasks.TaskMeta `json:",inline" yaml:",inline"`
	With           HTTPRequestInputs `json:"with" yaml:"with"`
}

// MakeHTTPRequestTask converts an spec to a task.
func MakeHTTPRequestTask(t *v2alpha2.TaskSpec) (tasks.Task, error) {
	if t.Task != LibraryName+"/"+HTTPRequestTaskName {
		return nil, fmt.Errorf("library and task need to be '%s' and '%s'", LibraryName, HTTPRequestTaskName)
	}
	var jsonBytes []byte
	var task tasks.Task
	// convert t to jsonBytes
	jsonBytes, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}
	// convert jsonString to ExecTask
	task = &HTTPRequestTask{}
	err = json.Unmarshal(jsonBytes, &task)
	return task, err
}

func (t *HTTPRequestTask) prepareRequest(ctx context.Context) (*http.Request, error) {
	tags := tasks.NewTags()
	exp, err := tasks.GetExperimentFromContext(ctx)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	obj, err := exp.ToMap()
	if err != nil {
		// error already logged by ToMap()
		// don't log it again
		return nil, err
	}

	// prepare for interpolation; add experiment as tag
	// Note that if versionRecommendedForPromotion is not set or there is no version corresponding to it,
	// then some placeholders may not be replaced
	tags = tags.
		With("this", obj).
		WithRecommendedVersionForPromotion(&exp.Experiment)

	secretName := t.With.Secret
	if secretName != nil {
		secret, err := tasks.GetSecret(*secretName)
		if err != nil {
			tags = tasks.WithSecret(secret)
		}
	}
	log.Info("final tags: ", tags)

	defaultMethod := v2alpha2.POSTMethodType
	method := t.With.Method
	if method == nil {
		method = &defaultMethod
	}
	log.Info("method: ", *method)

	defaultBody := ""
	body := t.With.Body
	if body != nil {
		if interpolated, err := tags.Interpolate(body); err == nil {
			body = &interpolated
		}
	} else {
		body = &defaultBody
	}
	log.Info("body:", *body)

	defaultAuthType := v2alpha2.AuthType("None")
	authType := t.With.AuthType
	if authType == nil {
		authType = &defaultAuthType
	}

	req, err := http.NewRequest(string(*method), t.With.URL, strings.NewReader(*body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-type", "application/json")
	for _, h := range t.With.Headers {
		hValue, err := tags.Interpolate(&h.Value)
		if err != nil {
			log.Warn("Unable to interpolate header "+h.Name, err)
		} else {
			req.Header.Set(h.Name, hValue)
		}
	}

	if *authType == v2alpha2.BasicAuthType {
		usernameTemplate := "{{ .secret.username }}"
		passwordTemplate := "{{ .secret.password }}"
		username, _ := tags.Interpolate(&usernameTemplate)
		password, _ := tags.Interpolate(&passwordTemplate)
		req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(username+":"+password)))
	} else if *authType == v2alpha2.BearerAuthType {
		tokenTemplate := "{{ .secret.token }}"
		token, _ := tags.Interpolate(&tokenTemplate)
		req.Header.Set("Authorization", "Bearer "+token)
	}

	return req, err
}

// Run the command.
func (t *HTTPRequestTask) Run(ctx context.Context) error {
	req, err := t.prepareRequest(ctx)

	if err != nil {
		return err
	}

	// send request
	var httpClient = &http.Client{
		Timeout: time.Second * 5,
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Error(err)
		return err
	}

	log.Info("RESPONSE STATUS: " + resp.Status)
	if resp.StatusCode >= 400 {

		err = errors.New(resp.Status)
		log.Error(err)
		return err
	}
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	log.Info(buf.String())

	return nil
}

package metrics

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/iter8-tools/etc3/api/v2alpha2"
	"github.com/iter8-tools/handler/base"
)

// Version contains header and url information needed to send requests to each version.
type Version struct {
	// name of the version
	// version names must be unique and must match one of the version names in the
	// VersionInfo field of the experiment
	Name string `json:"name" yaml:"name"`
	// how many queries per second will be sent to this version; optional; default 8
	QPS string `json:"qps,omitempty" yaml:"qps,omitempty"`
	// HTTP headers to use in the query for this version; optional
	Headers map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
	// URL to use for querying this version
	URL string `json:"url" yaml:"url"`
}

// CollectInputs contain the inputs to the metrics collection task to be executed.
type CollectInputs struct {
	// how long to run the metrics collector; optional; default 5s
	Time string `json:"time,omitempty" yaml:"time,omitempty"`
	// list of versions
	Versions []Version `json:"versions" yaml:"versions"`
	// URL of the JSON file to send during the query; optional
	PayloadURL string `json:"payloadURL,omitempty" yaml:"payloadURL,omitempty"`
	// HTTP method which can be GET or POST; optional; default GET
	Method string `json:"method,omitempty" yaml:"method,omitempty"`
}

// CollectTask enables collection of Iter8's built-in metrics.
type CollectTask struct {
	Library string        `json:"library" yaml:"library"`
	Task    string        `json:"task" yaml:"task"`
	With    CollectInputs `json:"with" yaml:"with"`
}

// Run executes a CollectTask
func (t *CollectTask) Run(ctx context.Context) error {
	log.Trace("collect task run started...")
	var err error
	return err
}

// MakeCollect converts a collect task spec into a CollectTask
func MakeCollect(t *v2alpha2.TaskSpec) (base.Task, error) {
	if t.Task != "metrics/collect" {
		return nil, errors.New("library and task need to be 'metrics' and 'collect'")
	}
	var err error
	var jsonBytes []byte
	var it base.Task
	// convert t to jsonBytes
	jsonBytes, err = json.Marshal(t)
	// convert jsonString to CollectTask
	if err == nil {
		ct := &CollectTask{}
		err = json.Unmarshal(jsonBytes, &ct)
	}
	return it, err
}

package metrics

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/iter8-tools/etc3/api/v2alpha2"
	"github.com/iter8-tools/handler/base"
	"github.com/iter8-tools/handler/utils"
)

// Version contains header and url information needed to send requests to each version.
type Version struct {
	// name of the version
	// version names must be unique and must match one of the version names in the
	// VersionInfo field of the experiment
	Name string `json:"name" yaml:"name"`
	// how many queries per second will be sent to this version; optional; default 8
	QPS *int32 `json:"qps,omitempty" yaml:"qps,omitempty"`
	// HTTP headers to use in the query for this version; optional
	Headers map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
	// URL to use for querying this version
	URL string `json:"url" yaml:"url"`
}

// CollectInputs contain the inputs to the metrics collection task to be executed.
type CollectInputs struct {
	// how long to run the metrics collector; optional; default 5s
	Time *string `json:"time,omitempty" yaml:"time,omitempty"`
	// list of versions
	Versions []Version `json:"versions" yaml:"versions"`
	// URL of the JSON file to send during the query; optional
	PayloadURL *string `json:"payloadURL,omitempty" yaml:"payloadURL,omitempty"`
	// HTTP method which can be GET or POST; optional; default GET
	Method *utils.HTTPMethod `json:"method,omitempty" yaml:"method,omitempty"`
}

// CollectTask enables collection of Iter8's built-in metrics.
type CollectTask struct {
	Library string        `json:"library" yaml:"library"`
	Task    string        `json:"task" yaml:"task"`
	With    CollectInputs `json:"with" yaml:"with"`
}

// Run executes a CollectTask
// figure out error handling every step of the way here ...
func (t *CollectTask) Run(ctx context.Context) error {
	log.Trace("collect task run started...")
	t.InitializeDefaults()
	var wg sync.WaitGroup
	fortioData := make(map[string]interface{})
	// lock ensures thread safety while updating fortioData from go routines
	var lock sync.Mutex
	// if errors occur in one of the parallel threads, errCh is used to communicate them
	errCh := make(chan error)
	defer close(errCh)

	for i := 0; i < len(t.With.Versions); i++ {
		for j := range t.With.Versions {
			// Increment the WaitGroup counter.
			wg.Add(1)
			// Launch a goroutine to fetch the Fortio data for this version.
			go func(k int) {
				// Decrement the counter when the goroutine completes.
				defer wg.Done()
				// Get Fortio data for version
				data, err := t.fortioDataForVersion(k)
				if err == nil {
					// Update fortioData
					lock.Lock()
					fortioData[t.With.Versions[k].Name] = data
					lock.Unlock()
				} else {
					errCh <- err
				}
			}(j)
		}
	}

	// See https://stackoverflow.com/questions/32840687/timeout-for-waitgroup-wait
	// Compute timeout as duration of fortio requests + 30s
	dur, err := time.ParseDuration(*t.With.Time)
	if err != nil {
		return err
	}
	if err = utils.WaitTimeoutOrError(&wg, dur+30*time.Second, errCh); err != nil {
		log.Error(err)
		return err
	} else {
		log.Trace("Wait group finished normally")
	}

	return nil
}

// fortioDataForVersion collects fortio data for a given version
func (t *CollectTask) fortioDataForVersion(j int) (map[string]interface{}, error) {
	return make(map[string]interface{}), nil
}

// InitializeDefaults sets the default values for optional fields that are empty
func (t *CollectTask) InitializeDefaults() {
	if t.With.Time == nil {
		t.With.Time = utils.StringPointer("5s")
	}
	if t.With.Method == nil {
		if t.With.PayloadURL == nil {
			t.With.Method = utils.HTTPMethodPointer(utils.GET)
		} else {
			t.With.Method = utils.HTTPMethodPointer(utils.POST)
		}
	}
	for i := 0; i < len(t.With.Versions); i++ {
		if t.With.Versions[i].QPS == nil {
			t.With.Versions[i].QPS = utils.Int32Pointer(8)
		}
	}
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

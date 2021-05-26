package metrics

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/iter8-tools/etc3/api/v2alpha2"
	"github.com/iter8-tools/handler/base"
	"github.com/iter8-tools/handler/utils"
	"k8s.io/apimachinery/pkg/api/resource"
)

// Version contains header and url information needed to send requests to each version.
type Version struct {
	// name of the version
	// version names must be unique and must match one of the version names in the
	// VersionInfo field of the experiment
	Name string `json:"name" yaml:"name"`
	// how many queries per second will be sent to this version; optional; default 8
	QPS *float32 `json:"qps,omitempty" yaml:"qps,omitempty"`
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
}

// CollectTask enables collection of Iter8's built-in metrics.
type CollectTask struct {
	Library string        `json:"library" yaml:"library"`
	Task    string        `json:"task" yaml:"task"`
	With    CollectInputs `json:"with" yaml:"with"`
}

type internalFortioResultSample struct {
	Start float64
	End   float64
	Count int
}

type internalDurationHistogram struct {
	Count int
	Min   float64
	Max   float64
	Sum   float64
	Avg   float64
	Data  []internalFortioResultSample
}

// internal struct for deserializing the result of a single Fortio run
type internalFortioResult struct {
	DurationHistogram internalDurationHistogram
	RetCodes          map[string]int
}

// FortioResultSample is a single item within the `Data` slice of FortioResult
type FortioResultSample struct {
	Start *resource.Quantity `json:"start" yaml:"start"`
	End   *resource.Quantity `json:"end" yaml:"end"`
	Count int                `json:"count" yaml:"count"`
}

// FortioResult is the go scheme for the per-version FortioResult stored in experiment resource
type FortioResult struct {
	Count    int                `json:"count" yaml:"count"`
	Min      *resource.Quantity `json:"min" yaml:"min"`
	Max      *resource.Quantity `json:"max" yaml:"max"`
	Sum      *resource.Quantity `json:"sum" yaml:"sum"`
	Avg      *resource.Quantity `json:"avg" yaml:"avg"`
	Data     FortioResultSample `json:"data" yaml:"data"`
	RetCodes map[string]int     `json:"retCodes" yaml:"retCodes"`
}

// getInternalFortioResult reads the contents from a Fortio output file and returns it in the internal format
func getInternalFortioResult(fortioOutputFile string) (*internalFortioResult, error) {
	// Open our jsonFile
	jsonFile, err := os.Open(fortioOutputFile)
	// if os.Open returns an error, handle it
	if err != nil {
		log.Error(err)
		return nil, err
	}
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	// read our opened jsonFile as a byte array.
	bytes, err := ioutil.ReadAll(jsonFile)
	// if ioutil.ReadAll returns an error, handle it
	if err != nil {
		log.Error(err)
		return nil, err
	}

	var ifr internalFortioResult
	err = json.Unmarshal(bytes, &ifr)
	// if json.Unmarshal returns an error, handle it
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return &ifr, nil

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

	// download JSON from URL -- if specified
	// this is intended to be used as a JSON payload file by Fortio
	tmpfileName := ""
	if t.With.PayloadURL != nil {
		var err error
		tmpfileName, err = payloadFile(*t.With.PayloadURL)
		if err != nil {
			return err
		}
	}
	defer os.Remove(tmpfileName) // clean up

	for j := range t.With.Versions {
		// Increment the WaitGroup counter.
		wg.Add(1)
		// Launch a goroutine to fetch the Fortio data for this version.
		go func(k int) {
			// Decrement the counter when the goroutine completes.
			defer wg.Done()
			// Get Fortio data for version
			data, err := t.fortioDataForVersion(k, tmpfileName)
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

	// See https://stackoverflow.com/questions/32840687/timeout-for-waitgroup-wait
	// Compute timeout as duration of fortio requests + 30s
	dur, err := time.ParseDuration(*t.With.Time)
	if err != nil {
		return err
	}
	if err = utils.WaitTimeoutOrError(&wg, dur+30*time.Second, errCh); err != nil {
		log.Error("Got error: ", err)
		return err
	} else {
		log.Trace("Wait group finished normally")
	}

	return nil
}

// payloadFile downloads JSON payload from a URL into a temp file, and returns its name
func payloadFile(url string) (string, error) {
	content, err := utils.GetJsonBytes(url)
	if err != nil {
		log.Error("Error while getting JSON bytes: ", err)
		return "", err
	}
	log.Trace("Got json bytes")

	tmpfile, err := ioutil.TempFile("/tmp", "payload.json")
	if err != nil {
		log.Fatal(err)
		return "", err
	}

	if _, err := tmpfile.Write(content); err != nil {
		tmpfile.Close()
		log.Fatal(err)
		return "", err
	}
	if err := tmpfile.Close(); err != nil {
		log.Fatal(err)
		return "", err
	}

	return tmpfile.Name(), nil
}

// fortioDataForVersion collects fortio data for a given version
func (t *CollectTask) fortioDataForVersion(j int, pf string) (*internalFortioResult, error) {
	var execOut bytes.Buffer
	// var errOut bytes.Buffer
	// append fortio subcommand
	args := []string{"load"}
	// append Fortio time flag
	args = append(args, "-t", *t.With.Time)
	// append Fortio qps flag
	args = append(args, "-qps", fmt.Sprintf("%f", *t.With.Versions[j].QPS))
	// append Fortio header flags
	for header, value := range t.With.Versions[j].Headers {
		args = append(args, "-H", fmt.Sprintf("%v: %v", header, value))
	}
	// download JSON payload -- if specified; and append
	if t.With.PayloadURL != nil {
		args = append(args, "-payload-file", pf)
	}

	// create json output file; and append
	jsonOutputFile, err := ioutil.TempFile("/tmp", "output.json.")
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	args = append(args, "-json", jsonOutputFile.Name())
	jsonOutputFile.Close()

	// append URL to be queried
	args = append(args, t.With.Versions[j].URL)

	// setup Fortio command
	cmd := exec.Command("fortio", args...)
	cmd.Stdout = &execOut
	cmd.Stderr = os.Stderr
	log.Trace("Invoking: " + cmd.String())

	// execute Fortio command
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	ifr, err := getInternalFortioResult(jsonOutputFile.Name())
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	log.Trace(ifr)
	return ifr, err
}

// InitializeDefaults sets the default values for optional fields in the collect task that are empty
func (t *CollectTask) InitializeDefaults() {
	if t.With.Time == nil {
		t.With.Time = utils.StringPointer("5s")
	}
	for i := 0; i < len(t.With.Versions); i++ {
		if t.With.Versions[i].QPS == nil {
			t.With.Versions[i].QPS = utils.Float32Pointer(8)
		}
	}
}

// MakeCollect constructs a CollectTask out of a collect task spec
func MakeCollect(t *v2alpha2.TaskSpec) (base.Task, error) {
	if t.Task != "metrics/collect" {
		return nil, errors.New("library and task need to be 'metrics' and 'collect'")
	}
	var err error
	var jsonBytes []byte
	var ct base.Task
	// convert t to jsonBytes
	jsonBytes, err = json.Marshal(t)
	// convert jsonString to CollectTask
	if err == nil {
		ct = &CollectTask{}
		err = json.Unmarshal(jsonBytes, &ct)
	}
	return ct, err
}

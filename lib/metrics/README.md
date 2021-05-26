[![Go Report Card](https://goreportcard.com/badge/github.com/iter8-tools/handler)](https://goreportcard.com/report/github.com/iter8-tools/handler)
[![Coverage](https://codecov.io/gh/iter8-tools/handler/branch/main/graphs/badge.svg?branch=main)](https://codecov.io/gh/iter8-tools/handler)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Reference](https://pkg.go.dev/badge/github.com/iter8-tools/handler.svg)](https://pkg.go.dev/github.com/iter8-tools/handler)
# Welcome to the Iter8 handler repo
> This repo enables Iter8 tasks

> Tasks are an extension mechanism for enhancing the behavior of Iter8 experiments and can be specified within the spec.strategy.actions field of the experiment. This repo provides Iter8 task implementations and the container image used for running these tasks during an experiment.

## Implementing an Iter8 task: `metrics/collect` example

Consider the `metrics/collect` task which collects Iter8's built-in metrics for specified versions. This task is implemented as follows.

### Create the `metrics` library 
1. Create the `lib/metrics` subfolder. Each task library corresponds to a subfolder under `lib`. This step is performed once per task library, when the first task within that library is being created.
2. Create the `metrics.go` file in the `lib/metrics` subfolder as follows. This file is the entry point for this task library. 
    ```go
    package metrics

    import (
      "errors"

      "github.com/iter8-tools/etc3/api/v2alpha2"
      "github.com/iter8-tools/handler/base"
      "github.com/iter8-tools/handler/utils"
      "github.com/sirupsen/logrus"
    )

    // Declare logger only once per package (in any file belonging to that package)
    var log *logrus.Logger

    func init() {
      // always use logger from utils
      // init logger once per package (in any file belonging to that package)
      log = utils.GetLogger()
    }

    // MakeTask constructs a Task from a TaskMeta or returns an error if any.
    func MakeTask(t *v2alpha2.TaskSpec) (base.Task, error) {
      switch t.Task {
      case "metrics/collect":
        return MakeCollect(t)
      default:
        return nil, errors.New("Unknown task: " + t.Task)
      }
    }
    ```

**Note the naming convention:** similar to `metrics.go` in the `lib/metrics` subfolder, there is also `common.go` in the `lib/common` subfolder. Every task library has a `lib/<libname>` subfolder containing a `<libname>.go` file in it.

### Stub the `metrics/collect` task
3. Create a file named `collect.go` in the `lib/metrics` subfolder.
4. Stub the task definition, its `Run` method, and its `Make` function as follows.
    ```go
    package metrics

    import (
        "context"
        "encoding/json"
        "errors"

        "github.com/iter8-tools/etc3/api/v2alpha2"
        "github.com/iter8-tools/handler/base"
        "github.com/iter8-tools/handler/utils"
        "github.com/sirupsen/logrus"
    )

    // Declare logger only once per package (in any file belonging to that package)
    var log *logrus.Logger

    func init() {
        // always use logger from utils
        // init logger once per package (in any file belonging to that package)
        log = utils.GetLogger()
    }

    // CollectTask enables collection of Iter8's built-in metrics.
    type CollectTask struct {
        Library string `json:"library" yaml:"library"`
        Task    string `json:"task" yaml:"task"`
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
    ```

**Note the naming convention:** similar to `collect.go` in the `lib/metrics` subfolder, there is also `exec.go` in the `lib/common` subfolder. Every task has a `<tasklibrary>/<taskname>.go` file.

### Import the `metrics` library in the task runner
5.  This step is performed once per task library, when the first task within that library is being created.
    * Open the file `run.go` in the `cmd` subfolder. 
    * Add `	"github.com/iter8-tools/handler/lib/metrics"` to the list of imported packages.
    * Change the `GetAction` function by extending its switch statement with this additional case statement:
        ```go
        // each task library corresponds to a case statement
        case "metrics":
            if action[i], err = metrics.MakeTask(&actionSpec[i]); err != nil {
                break Loop
            }
        ```

The task runner is now aware of the new library called `metrics`, and is capable of running the `metrics/collect` task (at present, the task does not do anything useful).

### Define input fields
6.  This task is intended to send requests and collect latency and error rate information for versions. Let us the input fields for the task that enables this goal.
    ```go
    ```

### Design the `Run` method
7. The `Run` method implements the execution logic for the task. We will start by identifying the high level steps in the method.
    * Initialize default values for empty optional fields
    * Use [Fortio](https://github.com/fortio/fortio) to generate requests for versions and collect latency and error metrics. Metrics collection for different versions should proceed in parallel.
    * If metrics collection for any of the versions failed, the task needs to exit. All failure / warning / other terminal messages are written to [termination logs](https://kubernetes.io/docs/tasks/debug-application-cluster/determine-reason-pod-failure/) for consumption by other components like etc3.
    * Read in the experiment object. Locally update experiment status with metrics collected for each version. If any old metrics data is available in the experiment status from previous runs of this task, the older metric values will be aggregated with the newly collected metric values.
    * Update experiment status in-cluster. Note that the experiment spec is not modified in-cluster.

### Implement the `Run` method



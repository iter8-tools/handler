package cmd

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/iter8-tools/etc3/api/v2alpha2"
	"github.com/iter8-tools/handler/base"
	"github.com/iter8-tools/handler/experiment"
	"github.com/iter8-tools/handler/lib/common"
	"github.com/iter8-tools/handler/lib/knative"
	"github.com/iter8-tools/handler/lib/metrics"
	"github.com/iter8-tools/handler/lib/notification"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/types"
)

// getExperimentNN gets the name and namespace of the experiment from environment variables.
// Returns error if unsuccessful.
func getExperimentNN() (*types.NamespacedName, error) {
	name := viper.GetViper().GetString("experiment_name")
	namespace := viper.GetViper().GetString("experiment_namespace")
	if len(name) == 0 || len(namespace) == 0 {
		return nil, errors.New("invalid experiment name/namespace")
	}
	return &types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, nil
}

// GetAction converts an action spec into an action.
func GetAction(exp *experiment.Experiment, actionSpec v2alpha2.Action) (base.Action, error) {
	action := make(base.Action, len(actionSpec))
	var err error
Loop:
	for i := 0; i < len(actionSpec); i++ {
		if actionSpecSubstr := strings.Split(actionSpec[i].Task, "/"); len(actionSpecSubstr) == 2 {
			switch actionSpecSubstr[0] {
			case "common":
				if action[i], err = common.MakeTask(&actionSpec[i]); err != nil {
					break Loop
				}
			case "knative":
				if action[i], err = knative.MakeTask(&actionSpec[i]); err != nil {
					break Loop
				}
			case notification.LibraryName:
				if action[i], err = notification.MakeTask(&actionSpec[i]); err != nil {
					// each task library corresponds to a case statement
					break Loop
				}
			case "metrics":
				if action[i], err = metrics.MakeTask(&actionSpec[i]); err != nil {
					break Loop
				}
			default:
				err = errors.New("unknown library: " + actionSpecSubstr[0])
			}
		} else {
			err = errors.New("no library specified")
		}
	}
	return action, err
}

// run is a helper function used in the definition of runCmd cobra command.
func run(cmd *cobra.Command, args []string) error {
	nn, err := getExperimentNN()
	if err == nil {
		var exp *experiment.Experiment
		if exp, err = (&experiment.Builder{}).FromCluster(nn).Build(); err == nil {
			var actionSpec v2alpha2.Action
			if actionSpec, err = exp.GetActionSpec(action); err == nil {
				var action base.Action
				if action, err = GetAction(exp, actionSpec); err == nil {
					ctx := context.WithValue(context.Background(), base.ContextKey("experiment"), exp)
					log.Trace("created context for experiment")
					err = action.Run(ctx)
					if err == nil {
						return nil
					}
				}
			} else {
				log.Error("could not find specified action: " + action)
				return nil
			}
		}
	}
	return err
}

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "run an action",
	Long:  `Sequentially execute all tasks in the specified action; if any task run results in an error, exit immediately with error.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := run(cmd, args); err != nil {
			log.Error("Exiting with error: ", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.PersistentFlags().StringVarP(&action, "action", "a", "", "name of the action")
	runCmd.MarkPersistentFlagRequired("action")
}

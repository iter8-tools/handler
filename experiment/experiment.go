// Package experiment enables construction of an experiment object with handler/task lists within it.
package experiment

import (
	"context"
	"errors"

	"github.com/iter8-tools/etc3/api/v2alpha2"
	iter8 "github.com/iter8-tools/etc3/api/v2alpha2"
	"github.com/iter8-tools/handler/base"
	"github.com/iter8-tools/handler/utils"
	"github.com/sirupsen/logrus"
)

var log *logrus.Logger

func init() {
	log = utils.GetLogger()
}

// Experiment is an enhancement of v2alpha2.Experiment struct with useful methods.
type Experiment struct {
	v2alpha2.Experiment
}

// Builder helps in construction of an experiment.
type Builder struct {
	err error
	exp *Experiment
}

// Build returns the built experiment or error.
// Must call FromFile or FromCluster on b prior to invoking Build.
func (b *Builder) Build() (*Experiment, error) {
	log.Trace(b)
	return b.exp, b.err
}

// GetExperimentFromContext gets the experiment object from given context.
func GetExperimentFromContext(ctx context.Context) (*Experiment, error) {
	//	ctx := context.WithValue(context.Background(), base.ContextKey("experiment"), e)
	if v := ctx.Value(base.ContextKey("experiment")); v != nil {
		log.Debug("found experiment")
		var e *Experiment
		var ok bool
		if e, ok = v.(*Experiment); !ok {
			return nil, errors.New("context has experiment value with wrong type")
		}
		return e, nil
	}
	return nil, errors.New("context has no experiment key")
}

// Interpolate interpolates input arguments based on tags of the version recommended for promotion in the experiment.
func (exp *Experiment) Interpolate(inputArgs []string) ([]string, error) {
	var recommendedBaseline string
	var args []string
	var err error
	if recommendedBaseline, err = exp.GetVersionRecommendedForPromotion(); err == nil {
		var versionDetail *iter8.VersionDetail
		if versionDetail, err = exp.GetVersionDetail(recommendedBaseline); err == nil {
			// get the tags
			tags := base.Tags{M: make(map[string]string)}
			tags.M["name"] = versionDetail.Name
			for i := 0; i < len(versionDetail.Variables); i++ {
				tags.M[versionDetail.Variables[i].Name] = versionDetail.Variables[i].Value
			}
			log.Trace(tags)
			args = make([]string, len(inputArgs))
			for i := 0; i < len(args); i++ {
				if args[i], err = tags.Interpolate(&inputArgs[i]); err != nil {
					break
				}
				log.Trace("input arg: ", inputArgs[i], " interpolated arg: ", args[i])
			}
		}
	}
	return args, err
}

/*
Copyright 2020 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package jobrunner

import (
	"context"
	"sync"

	"github.com/robfig/cron"
	"go.uber.org/zap"
	pingsourceinformer "knative.dev/eventing/pkg/client/injection/informers/sources/v1alpha2/pingsource"
	"knative.dev/eventing/pkg/kncloudevents"
	"knative.dev/eventing/pkg/tracing"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/source"
	tracingconfig "knative.dev/pkg/tracing/config"
)

const (
	// ReconcilerName is the name of the reconciler.
	ReconcilerName = "PingSources"

	// controllerAgentName is the string used by this controller to identify
	// itself when creating events.
	controllerAgentName = "ping-source-dispatcher"
)

// NewController initializes the controller and is called by the generated code.
// Registers event handlers to enqueue events.
func NewController(
	ctx context.Context,
	cmw configmap.Watcher,
) *controller.Impl {
	logger := logging.FromContext(ctx)

	// Setup trace publishing.
	iw := cmw.(*configmap.InformedWatcher)
	if err := tracing.SetupDynamicPublishing(logger, iw, "ping-source-dispatcher", tracingconfig.ConfigName); err != nil {
		logger.Fatalw("Error setting up trace publishing", zap.Error(err))
	}

	pingsourceInformer := pingsourceinformer.Get(ctx)

	r := &Reconciler{
		pingsourceLister: pingsourceInformer.Lister(),
		entryidMu:        sync.Mutex{},
		entryids:         make(map[string]cron.EntryID),
	}

	impl := controller.NewImpl(r, logger, ReconcilerName)

	logger.Info("Setting up event handlers")

	// Watch for pingsource objects
	pingsourceInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))

	// Create the cron job runner
	ceClient, err := kncloudevents.NewDefaultClient()
	if err != nil {
		logger.Fatalw("Error setting up trace publishing", zap.Error(err))
	}

	reporter, err := source.NewStatsReporter()
	if err != nil {
		logger.Error("error building statsreporter", zap.Error(err))
	}

	r.cronRunner = NewCronJobsRunner(ceClient, reporter, logger)

	// Start the cron job runner.
	go func() {
		err := r.cronRunner.Start(ctx.Done())
		if err != nil {
			logger.Error("Failed stopping the cron jobs runner.", zap.Error(err))
		}
	}()

	return impl
}

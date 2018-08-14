/*
Copyright 2018 The Knative Authors
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

package main

import (
	"flag"
	"log"

	"go.uber.org/zap"

	"github.com/knative/pkg/configmap"
	"github.com/knative/pkg/logging"
	"github.com/knative/pkg/logging/logkey"
	"github.com/knative/pkg/signals"
	"github.com/knative/pkg/webhook"

	v1alpha1channels "github.com/knative/eventing/pkg/apis/channels/v1alpha1"
	v1alpha1feeds "github.com/knative/eventing/pkg/apis/feeds/v1alpha1"
	v1alpha1flows "github.com/knative/eventing/pkg/apis/flows/v1alpha1"
	"github.com/knative/eventing/pkg/logconfig"
	"github.com/knative/eventing/pkg/system"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	flag.Parse()
	// Read the logging config and setup a logger.
	cm, err := configmap.Load("/etc/config-logging")
	if err != nil {
		log.Fatalf("Error loading logging configuration: %v", err)
	}
	config, err := logging.NewConfigFromMap(cm, logconfig.Webhook)
	if err != nil {
		log.Fatalf("Error parsing logging configuration: %v", err)
	}
	logger, atomicLevel := logging.NewLoggerFromConfig(config, logconfig.Webhook)
	defer logger.Sync()
	logger = logger.With(zap.String(logkey.ControllerType, logconfig.Webhook))

	logger.Info("Starting the Eventing Webhook")

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	clusterConfig, err := rest.InClusterConfig()
	if err != nil {
		logger.Fatal("Failed to get in cluster config", zap.Error(err))
	}

	kubeClient, err := kubernetes.NewForConfig(clusterConfig)
	if err != nil {
		logger.Fatal("Failed to get the client set", zap.Error(err))
	}

	// Watch the logging config map and dynamically update logging levels.
	configMapWatcher := configmap.NewDefaultWatcher(kubeClient, system.Namespace)

	configMapWatcher.Watch(logconfig.ConfigName, logging.UpdateLevelFromConfigMap(logger, atomicLevel, logconfig.Webhook, logconfig.Webhook))
	if err = configMapWatcher.Start(stopCh); err != nil {
		logger.Fatalf("failed to start webhook configmap watcher: %v", err)
	}

	options := webhook.ControllerOptions{
		ServiceName:    "webhook",
		DeploymentName: "webhook",
		Namespace:      system.Namespace,
		Port:           443,
		SecretName:     "webhook-certs",
		WebhookName:    "webhook.eventing.knative.dev",
	}
	controller := webhook.AdmissionController{
		Client:  kubeClient,
		Options: options,
		Handlers: map[schema.GroupVersionKind]runtime.Object{
			// For group channels.knative.dev,
			v1alpha1channels.SchemeGroupVersion.WithKind("Bus"):          &v1alpha1channels.Bus{},
			v1alpha1channels.SchemeGroupVersion.WithKind("ClusterBus"):   &v1alpha1channels.ClusterBus{},
			v1alpha1channels.SchemeGroupVersion.WithKind("Channel"):      &v1alpha1channels.Channel{},
			v1alpha1channels.SchemeGroupVersion.WithKind("Subscription"): &v1alpha1channels.Subscription{},

			// For group feeds.knative.dev,
			v1alpha1feeds.SchemeGroupVersion.WithKind("EventSource"):        &v1alpha1feeds.EventSource{},
			v1alpha1feeds.SchemeGroupVersion.WithKind("ClusterEventSource"): &v1alpha1feeds.ClusterEventSource{},
			v1alpha1feeds.SchemeGroupVersion.WithKind("EventType"):          &v1alpha1feeds.EventType{},
			v1alpha1feeds.SchemeGroupVersion.WithKind("ClusterEventType"):   &v1alpha1feeds.ClusterEventType{},
			v1alpha1feeds.SchemeGroupVersion.WithKind("Feed"):               &v1alpha1feeds.Feed{},

			// For group flows.knative.dev,
			v1alpha1flows.SchemeGroupVersion.WithKind("Flow"): &v1alpha1flows.Flow{},
		},
		Logger: logger,
	}
	if err != nil {
		logger.Fatal("Failed to create the admission controller", zap.Error(err))
	}
	controller.Run(stopCh)
}

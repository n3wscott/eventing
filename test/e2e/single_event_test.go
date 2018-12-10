// +build e2e

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

package e2e

import (
	"fmt"
	"testing"

	"github.com/knative/eventing/test"
	pkgTest "github.com/knative/pkg/test"
	"github.com/knative/pkg/test/logging"
	"k8s.io/apimachinery/pkg/util/uuid"
)

const (
	channelName         = "e2e-singleevent-channel"
	provisionerName     = "in-memory-channel"
	routeName           = "e2e-singleevent-subscriber"
	senderName          = "e2e-singleevent-sender"
	subscriberImageName = "logevents"
	subscriptionName    = "e2e-singleevent-subscription"
)

func TestSingleEvent(t *testing.T) {
	logger := logging.GetContextLogger("TestSingleEvent")

	clients, cleaner := Setup(t, logger)
	defer TearDown(clients, cleaner, logger)

	logger.Infof("Creating Route and Config")
	// The receiver of events which is accessible through Route
	configImagePath := test.ImagePath("logevents")
	if err := WithRouteReady(clients, logger, cleaner, routeName, configImagePath); err != nil {
		t.Fatalf("The Route was not marked as Ready to serve traffic: %v", err)
	}

	logger.Infof("Creating Channel and Subscription")
	channel := test.Channel(channelName, pkgTest.Flags.Namespace, test.ClusterChannelProvisioner(provisionerName))
	sub := test.Subscription(subscriptionName, pkgTest.Flags.Namespace, test.ChannelRef(channelName), test.SubscriberSpecForRoute(routeName), nil)

	if err := WithChannelAndSubscriptionReady(clients, channel, sub, logger, cleaner); err != nil {
		t.Fatalf("The Channel or Subscription were not marked as Ready: %v", err)
	}

	logger.Infof("Creating event sender")
	body := fmt.Sprintf("TestSingleEvent %s", uuid.NewUUID())
	event := test.CloudEvent{
		Data: fmt.Sprintf(`{"msg":%q}`, body),
	}
	pod := test.EventSenderPod(senderName, pkgTest.Flags.Namespace, channel.Status.Address.Hostname, event)
	if err := CreatePod(clients, pod, logger, cleaner); err != nil {
		t.Fatalf("Failed to create event sender pod: %v", err)
	}

	if err := WaitForAllPodsRunning(clients, logger, pkgTest.Flags.Namespace); err != nil {
		t.Fatalf("Error waiting for pods to become running: %v", err)
	}

	if err := WaitForLogContent(clients, logger, routeName, "sendevent", body); err != nil {
		t.Fatalf("String %q not found in logs of receiver %q: %v", body, routeName, err)
	}
}

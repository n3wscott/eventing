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
	channelName      = "e2e-singleevent"
	provisionerName  = "in-memory-channel"
	subscriberName   = "e2e-singleevent-subscriber"
	senderName       = "e2e-singleevent-sender"
	subscriptionName = "e2e-singleevent-subscription"
)

func TestSingleEvent(t *testing.T) {
	logger := logging.GetContextLogger("TestSingleEvent")
	ns := pkgTest.Flags.Namespace

	clients, cleaner := Setup(t, logger)
	defer TearDown(clients, cleaner, logger)

	logger.Infof("Creating Logger Pod and Service")
	selector := map[string]string{"e2etest": uuid.NewUUID()}
	subscriberPod := test.EventLoggerPod(routeName, ns, selector)
	if err := CreatePod(clients, subscriberPod, logger, cleaner); err != nil {
		t.Fatalf("Failed to create event logger pod: %v", err)
	}
	//TODO can use pod ip?
	subscriberSvc := test.Service(routeName, ns, selector)
	if err := CreateService(clients, subscriberSvc, logger, cleaner); err != nil {
		t.Fatalf("Failed to create event logger service: %v", err)
	}

	logger.Infof("Creating Channel and Subscription")
	channel := test.Channel(channelName, ns, test.ClusterChannelProvisioner(provisionerName))
	logger.Infof("channel: %#v", channel)
	sub := test.Subscription(subscriptionName, ns, test.ChannelRef(channelName), test.SubscriberSpecForRoute(routeName), nil)
	logger.Infof("sub: %#v", sub)

	if err := WithChannelAndSubscriptionReady(clients, channel, sub, logger, cleaner); err != nil {
		t.Fatalf("The Channel or Subscription were not marked as Ready: %v", err)
	}

	logger.Infof("Creating event sender")
	body := fmt.Sprintf("TestSingleEvent %s", uuid.NewUUID())
	event := test.CloudEvent{
		Type: "test.eventing.knative.dev",
		Data: fmt.Sprintf(`{"msg":%q}`, body),
	}
	url := fmt.Sprintf("http://%s", channel.Status.Address.Hostname)
	pod := test.EventSenderPod(senderName, ns, url, event)
	logger.Infof("sender pod: %#v", pod)
	if err := CreatePod(clients, pod, logger, cleaner); err != nil {
		t.Fatalf("Failed to create event sender pod: %v", err)
	}

	if err := WaitForAllPodsRunning(clients, logger, ns); err != nil {
		t.Fatalf("Error waiting for pods to become running: %v", err)
	}
	logger.Infof("All pods running")

	if err := WaitForLogContent(clients, logger, routeName, loggerPod.Spec.Containers[0].Name, ns, body); err != nil {
		t.Fatalf("String %q not found in logs of logger pod %q: %v", body, routeName, err)
	}
}

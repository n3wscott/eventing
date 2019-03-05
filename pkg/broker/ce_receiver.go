/*
 * Copyright 2019 The Knative Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package broker

import (
	"context"
	"errors"
	"fmt"
	"github.com/cloudevents/sdk-go/pkg/cloudevents"
	ceclient "github.com/cloudevents/sdk-go/pkg/cloudevents/client"
	cecontext "github.com/cloudevents/sdk-go/pkg/cloudevents/context"
	cehttp "github.com/cloudevents/sdk-go/pkg/cloudevents/transport/http"
	"net/http"
	"net/url"

	eventingv1alpha1 "github.com/knative/eventing/pkg/apis/eventing/v1alpha1"
	"github.com/knative/eventing/pkg/provisioners"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultPort = 8080
)

// Receiver parses Cloud Events, determines if they pass a filter, and sends them to a subscriber.
type Receiver struct {
	logger *zap.Logger
	client client.Client

	port       int
	httpClient *http.Client

	ce ceclient.Client
}

// New creates a new Receiver and its associated MessageReceiver. The caller is responsible for
// Start()ing the returned MessageReceiver.
func New(logger *zap.Logger, client client.Client) *Receiver {

	r := &Receiver{
		logger:     logger,
		client:     client,
		port:       defaultPort,
		httpClient: &http.Client{},
	}
	return r
}

// Start begins to receive messages for the receiver.
//
// Only HTTP POST requests to the root path (/) are accepted. If other paths or
// methods are needed, use the HandleRequest method directly with another HTTP
// server.
//
// This method will block until a message is received on the stop channel.
func (r *Receiver) Start(stopCh <-chan struct{}) error {
	if err := r.start(); err != nil {
		return err
	}

	<-stopCh

	// TODO: stop the ce client.
	return nil
}

func (r *Receiver) start() error {

	if r.ce == nil {
		c, err := ceclient.NewHTTPClient(
			ceclient.WithHTTPPort(r.port),
			//ceclient.WithHTTPBinaryEncoding(),  // <-- use if you want to not change the incoming cloudevent versions
			ceclient.WithHTTPEncoding(cehttp.BinaryV01), // <-- forces all output to be v0.1
			ceclient.WithHTTPClient(r.httpClient),
		)
		if err != nil {
			return fmt.Errorf("failed to create cloudevents http client, %s", err.Error())
		}
		r.ce = c
	}

	if err := r.ce.StartReceiver(context.TODO(), r.Receive); err != nil {
		return fmt.Errorf("failed to start cloudevents receiver, %s", err.Error())
	}

	return nil
}

func (r *Receiver) Receive(event cloudevents.Event) (*cloudevents.Event, error) {
	var tctx *cehttp.TransportContext
	if event.TransportContext != nil {
		var ok bool
		if tctx, ok = event.TransportContext.(*cehttp.TransportContext); !ok {
			r.logger.Error("Unable to cast TransportContext from event.")
			return nil, fmt.Errorf("unable to get transport context from event")
		}
	}
	triggerRef, err := provisioners.ParseChannel(tctx.Host)
	if err != nil {
		r.logger.Error("Unable to parse host as a trigger", zap.Error(err), zap.String("host", tctx.Host))
		return nil, fmt.Errorf("bad host")
	}

	responseEvent, err := r.sendEvent(triggerRef, event)
	if err != nil {
		r.logger.Error("Error sending the event", zap.Error(err))
		return nil, err
	}

	return responseEvent, err
}

// sendEvent sends an event to a subscriber if the trigger filter passes.
func (r *Receiver) sendEvent(trigger provisioners.ChannelReference, event cloudevents.Event) (*cloudevents.Event, error) {
	r.logger.Debug("Received message", zap.Any("triggerRef", trigger))
	ctx := context.Background()

	t, err := r.getTrigger(ctx, trigger)
	if err != nil {
		r.logger.Info("Unable to get the Trigger", zap.Error(err), zap.Any("triggerRef", trigger))
		return nil, err
	}

	subscriberURIString := t.Status.SubscriberURI
	if subscriberURIString == "" {
		r.logger.Error("Unable to read subscriberURI")
		return nil, errors.New("unable to read subscriberURI")
	}
	if _, err := url.Parse(subscriberURIString); err != nil {
		r.logger.Error("Unable to parse subscriberURI", zap.Error(err), zap.String("subscriberURIString", subscriberURIString))
		return nil, err
	}

	if !r.shouldSendMessage(&t.Spec, event) {
		r.logger.Debug("Message did not pass filter", zap.Any("triggerRef", trigger))
		return nil, nil
	}

	cxctx := cecontext.ContextWithTarget(context.TODO(), subscriberURIString)
	return r.ce.Send(cectx, event)
}

// Initialize the client. Mainly intended to load stuff in its cache.
func (r *Receiver) initClient() error {
	// We list triggers so that we can load the client's cache. Otherwise, on receiving an event, it
	// may not find the trigger and would return an error.
	opts := &client.ListOptions{
		// Set Raw because if we need to get more than one page, then we will put the continue token
		// into opts.Raw.Continue.
		Raw: &metav1.ListOptions{},
	}
	for {
		tl := &eventingv1alpha1.TriggerList{}
		if err := r.client.List(context.TODO(), opts, tl); err != nil {
			return err
		}
		if tl.Continue != "" {
			opts.Raw.Continue = tl.Continue
		} else {
			break
		}
	}
	return nil
}

func (r *Receiver) getTrigger(ctx context.Context, ref provisioners.ChannelReference) (*eventingv1alpha1.Trigger, error) {
	t := &eventingv1alpha1.Trigger{}
	err := r.client.Get(ctx,
		types.NamespacedName{
			Namespace: ref.Namespace,
			Name:      ref.Name,
		},
		t)
	return t, err
}

// shouldSendMessage determines whether message 'm' should be sent based on the triggerSpec 'ts'.
// Currently it supports exact matching on type and/or source of events.
func (r *Receiver) shouldSendMessage(ts *eventingv1alpha1.TriggerSpec, event cloudevents.Event) bool {
	if ts.Filter == nil || ts.Filter.SourceAndType == nil {
		r.logger.Error("No filter specified")
		return false
	}
	filterType := ts.Filter.SourceAndType.Type
	if filterType != eventingv1alpha1.TriggerAnyFilter && filterType != event.Type() {
		r.logger.Debug("Wrong type", zap.String("trigger.spec.filter.sourceAndType.type", filterType), zap.String("event.Type()", event.Type()))
		return false
	}
	filterSource := ts.Filter.SourceAndType.Source
	s := event.Context.AsV01().Source
	actualSource := s.String()
	//actualSource := event.Context.AsV01().Source.String()
	if filterSource != eventingv1alpha1.TriggerAnyFilter && filterSource != actualSource {
		r.logger.Debug("Wrong source", zap.String("trigger.spec.filter.sourceAndType.source", filterSource), zap.String("message.source", actualSource))
		return false
	}
	return true
}

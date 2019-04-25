/*
Copyright 2019 The Knative Authors

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

package testing

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/knative/eventing/pkg/apis/sources/v1alpha1"
)

// ContainerSourceOption enables further configuration of a CronJob.
type ContainerSourceOption func(*v1alpha1.ContainerSource)

// NewCronJob creates a CronJob with CronJobOptions
func NewContainerSource(name, namespace string, o ...ContainerSourceOption) *v1alpha1.ContainerSource {
	c := &v1alpha1.ContainerSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	for _, opt := range o {
		opt(c)
	}
	//c.SetDefaults(context.Background()) // TODO: We should add defaults and validation.
	return c
}

// WithInitContainerSourceConditions initializes the ContainerSource's conditions.
func WithInitContainerSourceConditions(s *v1alpha1.ContainerSource) {
	s.Status.InitializeConditions()
}

func WithContainerSourceSinkNotFound(msg string) ContainerSourceOption {
	return func(s *v1alpha1.ContainerSource) {
		s.Status.MarkNoSink("NotFound", msg)
	}
}

func WithContainerSourceSink(uri string) ContainerSourceOption {
	return func(s *v1alpha1.ContainerSource) {
		s.Status.MarkSink(uri)
	}
}

func WithContainerSourceDeployed(s *v1alpha1.ContainerSource) {
	s.Status.MarkDeployed()
}

func WithContainerSourceDeleted(c *v1alpha1.ContainerSource) {
	t := metav1.NewTime(time.Unix(1e9, 0))
	c.ObjectMeta.SetDeletionTimestamp(&t)
}

func WithContainerSourceSpec(spec v1alpha1.ContainerSourceSpec) ContainerSourceOption {
	return func(c *v1alpha1.ContainerSource) {
		c.Spec = spec
	}
}

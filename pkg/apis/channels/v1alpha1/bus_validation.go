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

package v1alpha1

import (
	"fmt"
	"github.com/knative/pkg/apis"
	"k8s.io/apimachinery/pkg/util/validation"
	"strings"
)

func (b *Bus) Validate() *apis.FieldError {
	return b.Spec.Validate().ViaField("spec")
}

func (bs *BusSpec) Validate() *apis.FieldError {
	if bs.Parameters != nil {
		return bs.Parameters.Validate().ViaField("parameters")
	}
	return nil
}

func (bp *BusParameters) Validate() *apis.FieldError {
	if bp.Channel != nil {
		for i, p := range *bp.Channel {
			errs := validation.IsConfigMapKey(p.Name)
			if len(errs) > 0 {
				return ErrInvalidParameterName("name", errs).ViaField(fmt.Sprintf("channel[%d]", i))
			}
		}
	}
	if bp.Subscription != nil {
		for i, p := range *bp.Subscription {
			errs := validation.IsConfigMapKey(p.Name)
			if len(errs) > 0 {
				return ErrInvalidParameterName("name", errs).ViaField(fmt.Sprintf("subscription[%d]", i))
			}
		}
	}
	return nil
}

func (current *Bus) CheckImmutableFields(og apis.Immutable) *apis.FieldError {
	// TODO(n3wscott): Anything to check?
	return nil
}

// TODO: use from pkg when https://github.com/knative/pkg/pull/34 lands
func ErrInvalidParameterName(path string, errs []string) *apis.FieldError {
	return &apis.FieldError{
		Message: fmt.Sprintf("invalid parameter name"),
		Paths:   []string{path},
		Details: strings.Join(errs, ", "),
	}
}

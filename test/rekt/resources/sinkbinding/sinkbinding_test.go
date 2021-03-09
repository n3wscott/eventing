/*
Copyright 2021 The Knative Authors

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

package sinkbinding_test

import (
	"os"

	"knative.dev/reconciler-test/pkg/manifest"
)

// The following examples validate the processing of the With* helper methods
// applied to config and go template parser.

func Example_min() {
	images := map[string]string{}
	cfg := map[string]interface{}{
		"name":      "foo",
		"namespace": "bar",
		"sink": map[string]interface{}{
			"ref": map[string]interface{}{
				"kind":       "AKind",
				"apiVersion": "something.valid/v1",
				"name":       "thesink",
			},
		},
		"subject": map[string]interface{}{
			"kind":       "BKind",
			"apiVersion": "interesting/v1",
			"name":       "thesubject",
		},
	}

	files, err := manifest.ExecuteLocalYAML(images, cfg)
	if err != nil {
		panic(err)
	}

	manifest.OutputYAML(os.Stdout, files)
	// Output:
	// apiVersion: sources.knative.dev/v1
	// kind: SinkBinding
	// metadata:
	//   name: foo
	//   namespace: bar
	// spec:
	//   sink:
	//     ref:
	//       apiVersion: something.valid/v1
	//       kind: AKind
	//       namespace: bar
	//       name: thesink
	//   subject:
	//     kind: BKind
	//     apiVersion: interesting/v1
	//     namespace: bar
	//     name: thesubject
}

func Example_full() {
	images := map[string]string{}
	cfg := map[string]interface{}{
		"name":      "foo",
		"namespace": "bar",
		"sink": map[string]interface{}{
			"ref": map[string]string{
				"kind":       "sinkkind",
				"namespace":  "sinknamespace",
				"name":       "sinkname",
				"apiVersion": "sinkversion",
			},
			"uri": "uri/parts",
		},
	}

	files, err := manifest.ExecuteLocalYAML(images, cfg)
	if err != nil {
		panic(err)
	}

	manifest.OutputYAML(os.Stdout, files)
	// Output:
	// apiVersion: sources.knative.dev/v1beta2
	// kind: PingSource
	// metadata:
	//   name: foo
	//   namespace: bar
	// spec:
	//   schedule: "*/1 * * * *"
	//   contentType: "application/json"
	//   data: '{"message": "Hello world!"}'
	//   sink:
	//     ref:
	//       kind: sinkkind
	//       namespace: bar
	//       name: sinkname
	//       apiVersion: sinkversion
	//     uri: uri/parts
}

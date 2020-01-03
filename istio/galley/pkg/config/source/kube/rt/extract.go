// Copyright 2019 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rt

import (
	"github.com/gogo/protobuf/proto"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"istio.io/istio/galley/pkg/config/meta/schema"
	"istio.io/istio/galley/pkg/config/resource"
)

// ToResource converts the given object and proto to a resource.Instance
func ToResource(object metav1.Object, r *schema.KubeResource, item proto.Message) *resource.Instance {
	var o *Origin

	name := resource.NewFullName(resource.Namespace(object.GetNamespace()), resource.LocalName(object.GetName()))
	version := resource.Version(object.GetResourceVersion())

	if r != nil {
		o = &Origin{
			FullName:   name,
			Collection: r.Collection.Name,
			Kind:       r.Kind,
			Version:    version,
		}
	}

	return &resource.Instance{
		Metadata: resource.Metadata{
			FullName:    name,
			Version:     version,
			Annotations: object.GetAnnotations(),
			Labels:      object.GetLabels(),
			CreateTime:  object.GetCreationTimestamp().Time,
		},
		Message: item,
		Origin:  o,
	}
}

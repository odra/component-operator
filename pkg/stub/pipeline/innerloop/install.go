/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package innerloop

import (
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/operator-framework/operator-sdk/pkg/util/k8sutil"
	log "github.com/sirupsen/logrus"
	"github.com/snowdrop/component-operator/pkg/apis/component/v1alpha1"
	"github.com/snowdrop/component-operator/pkg/stub/pipeline"
	"github.com/snowdrop/component-operator/pkg/util/kubernetes"
	util "github.com/snowdrop/component-operator/pkg/util/template"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"text/template"
)

// NewInstallStep creates a step that handles the creation of the DeploymentConfig
func NewInstallStep() pipeline.Step {
	return &installStep{}
}

type installStep struct{}

func (installStep) Name() string {
	return "deploy"
}

func (installStep) CanHandle(component *v1alpha1.Component) bool {
	return component.Status.Phase == ""
}

func (installStep) Handle(component *v1alpha1.Component, deleted bool) error {
	target := component.DeepCopy()
	return installInnerLoop(target)
}

func installInnerLoop(component *v1alpha1.Component) error {
	// Get Current Namespace
	namespace, err := kubernetes.GetClientCurrentNamespace("")
	if err != nil {
		return err
	}
	component.ObjectMeta.Namespace = namespace

	// TODO Add a key to get the templates associated to a category such as : innerloop, ....
	for _, tmpl := range util.Templates {
		if strings.HasPrefix(tmpl.Name(), "innerloop") {

			switch tmpl.Name() {
			case "innerloop/imagestream":
				// Get supervisord imagestream
				component.Spec.Images = GetSupervisordImage()

				// Define the key of the image to search accoring to the runtime
				imageKey := ""
				switch r := component.Spec.Runtime; r {
				case "spring-boot", "vert.x", "thornthail":
					imageKey = "java"
				case "nodejs":
					imageKey = "nodejs"
				default:
					imageKey = "java"
				}

				// Append runtime's image
				component.Spec.Images = append(component.Spec.Images, CreateTypeImage(true, "dev-runtime", "latest", image[imageKey], false))

				err := createResource(tmpl, component)
				if err != nil {
					return err
				}

				log.Infof("#### Created 'supervisord and '%s' imagestreams", image[imageKey])

			case "innerloop/pvc":
				component.Spec.Storage.Name = "m2-data"
				component.Spec.Storage.Capacity = "1Gi"
				component.Spec.Storage.Mode = "ReadWriteOnce"
				err := createResource(tmpl, component)
				if err != nil {
					return err
				}
				log.Infof("#### Created '%s' persistent volume storage; capacity: '%s'; mode '%s'", component.Spec.Storage.Name, component.Spec.Storage.Capacity, component.Spec.Storage.Mode)

			case "innerloop/deploymentconfig":
				if component.Spec.Port == 0 {
					component.Spec.Port = 8080 // Add a default port if empty
				}
				component.Spec.SupervisordName = "copy-supervisord"

				// Enrich Env Vars with Default values
				populateEnvVar(component)

				err := createResource(tmpl, component)
				if err != nil {
					return err
				}
				log.Infof("#### Created dev's deployment config containing as initContainer : supervisord")

			case "innerloop/route":
				if component.Spec.ExposeService {
					err := createResource(tmpl, component)
					if err != nil {
						return err
					}
					log.Infof("#### Exposed service's port '%d' as cluster's route",component.Spec.Port)
				}

			case "innerloop/service":
				if component.Spec.Port == 0 {
					component.Spec.Port = 8080 // Add a default port if empty
				}
				err := createResource(tmpl, component)
				if err != nil {
					return err
				}
				log.Infof("#### Created service's port '%d'",component.Spec.Port)

			default:
				err := createResource(tmpl, component)
				if err != nil {
					return err
				}
			}
		}
	}
	log.Infof("#### Created %s CRD's component ", component.Name)
	component.Status.Phase = v1alpha1.PhaseDeploying

	err = sdk.Update(component)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}

	log.Info("### Pipeline 'innerloop' ended ###")

	return nil
}

func createResource(tmpl template.Template, component *v1alpha1.Component) error {
	res, err := newResourceFromTemplate(tmpl, component)
	if err != nil {
		return err
	}

	for _, r := range res {
		err = sdk.Create(r)
		if err != nil && !k8serrors.IsAlreadyExists(err) {
			return err
		}
	}

	return nil
}

func newResourceFromTemplate(template template.Template, component *v1alpha1.Component) ([]runtime.Object, error) {
	var result = []runtime.Object{}

	var b = util.Parse(template, component)
	r, err := kubernetes.PopulateKubernetesObjectFromYaml(b.String())
	if err != nil {
		return nil, err
	}

	if strings.HasSuffix(r.GetKind(), "List") {
		l, err := r.ToList()
		if err != nil {
			return nil, err
		}
		for _, item := range l.Items {
			obj, err := k8sutil.RuntimeObjectFromUnstructured(&item)
			if err != nil {
				return nil, err
			}

			kubernetes.SetNamespaceAndOwnerReference(obj, component)
			result = append(result, obj)
		}
	} else {
		obj, err := k8sutil.RuntimeObjectFromUnstructured(r)
		if err != nil {
			return nil, err
		}

		kubernetes.SetNamespaceAndOwnerReference(obj, component)
		result = append(result, obj)
	}
	return result, nil
}

func getLabels(component string) map[string]string {
	labels := map[string]string{
		"Component": component,
	}
	return labels
}

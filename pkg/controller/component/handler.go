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

package component

import (
	"context"
	"github.com/snowdrop/component-operator/pkg/apis/component/v1alpha1"
	"github.com/snowdrop/component-operator/pkg/stub/pipeline"
	"github.com/snowdrop/component-operator/pkg/stub/pipeline/innerloop"
	"github.com/snowdrop/component-operator/pkg/stub/pipeline/link"
	"github.com/snowdrop/component-operator/pkg/stub/pipeline/servicecatalog"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	log "github.com/sirupsen/logrus"

)

// Add creates a new AppService Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileComponent{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
		innerLoopSteps: []pipeline.Step{
			innerloop.NewInstallStep(),
		},
		serviceCatalogSteps: []pipeline.Step{
			servicecatalog.NewServiceInstanceStep(),
		},
		linkSteps: []pipeline.Step{
			link.NewLinkStep(),
		},
	}
}

type ReconcileComponent struct {
	client              client.Client
	scheme              *runtime.Scheme
	innerLoopSteps      []pipeline.Step
	serviceCatalogSteps []pipeline.Step
	linkSteps           []pipeline.Step
}

func (r *ReconcileComponent) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	operation := ""
	// Fetch the AppService instance
	component := &v1alpha1.Component{}
	err := r.client.Get(context.TODO(), request.NamespacedName, component)
	if err != nil {
		if errors.IsNotFound(err) {
			operation = "deleted"
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Printf("Reconciling AppService %s/%s - operation %s\n", request.Namespace, request.Name, operation)
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Printf("Reconciling AppService %s/%s - operation %s\n", request.Namespace, request.Name, operation)

		return reconcile.Result{}, err
	}
	// appService instance created
	operation = "created"
	for _, a := range r.innerLoopSteps {
		if a.CanHandle(component) {
			if err := a.Handle(component); err != nil {
				return reconcile.Result{}, err
			}
		}
	}

	log.Printf("Reconciling AppService %s/%s - operation %s\n", request.Namespace, request.Name, operation)
	return reconcile.Result{}, nil
}




/*func (h *Handler) Handle(ctx context.Context, event sdk.Event) error {
	switch o := event.Object.(type) {

	case *v1alpha1.Export:
		if o.Spec.Name != "" {
			log.Info("### Invoking export on ", o.Name)
		}

	case *v1alpha1.Component:
		// Check the DeploymentMode to install the component/runtime
		if o.Spec.Runtime != "" && o.Spec.DeploymentMode == "innerloop" {
			// log.Debug("DeploymentMode :", o.Spec.DeploymentMode)
			for _, a := range h.innerLoopSteps {
				action := ""
				if a.CanHandle(o) {
					if event.Deleted {
						action = "deleted"
					} else {
						action = a.Name()
					}
					log.Infof("### Invoking pipeline 'innerloop', action '%s' on %s", action, o.Name)
					if err := a.Handle(o, event.Deleted); err != nil {
						return err
					}
				}
			}
			break
		}
		// Check if the component is a service
		if o.Spec.Services != nil {
			removeServiceInstanceStep := servicecatalog.RemoveServiceInstanceStep()
			if event.Deleted {
				log.Infof("### Invoking'service catalog', action '%s' on %s", "delete", o.Name)
				if err := removeServiceInstanceStep.Handle(o, event.Deleted); err != nil {
					return err
				}
			}
			for _, a := range h.serviceCatalogSteps {
				if a.CanHandle(o) {
					log.Infof("### Invoking'service catalog', action '%s' on %s", a.Name(), o.Name)
					if err := a.Handle(o, event.Deleted); err != nil {
						return err
					}
				}
			}
			break
		}
		// Check if the component is a Link
		if o.Spec.Link != nil {
			for _, a := range h.linkSteps {
				action := ""
				if a.CanHandle(o) {
					if event.Deleted {
						action = "deleted"
					} else {
						action = a.Name()
					}
					log.Infof("### Invoking'link', action '%s' on %s", action, o.Name)
					if err := a.Handle(o, event.Deleted); err != nil {
						return err
					}
				}
			}
			break
		}
	}
	return nil
}*/

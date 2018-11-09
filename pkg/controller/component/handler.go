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
	"github.com/snowdrop/component-operator/pkg/pipeline"
	"github.com/snowdrop/component-operator/pkg/pipeline/innerloop"
	"github.com/snowdrop/component-operator/pkg/pipeline/link"
	"github.com/snowdrop/component-operator/pkg/pipeline/servicecatalog"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

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

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("component-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource AppService
	err = c.Watch(&source.Kind{Type: &v1alpha1.Component{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileComponent{}

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
			// Component has been deleted
			operation = "deleted"

			// Check if the component is a Service and then delete the ServiceInstance, ServiceBinding
			if component.Spec.Services != nil {
				removeServiceInstanceStep := servicecatalog.RemoveServiceInstanceStep()
					log.Infof("### Invoking'service catalog', action '%s' on %s", "delete", component.Name)
					if err := removeServiceInstanceStep.Handle(component, &r.client); err != nil {
						log.Error(err)
					}
			}

			log.Printf("Reconciling AppService %s/%s - operation %s\n", request.Namespace, request.Name, operation)
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Printf("Reconciling AppService %s/%s - operation %s\n", request.Namespace, request.Name, operation)
		return reconcile.Result{}, err
	}

	// Component Custom Resource instance has been created
	operation = "created"

	// Check if Spec is not null and if the DeploymentMode strategy is equal to innerloop
	if component.Spec.Runtime != "" && component.Spec.DeploymentMode == "innerloop" {
		for _, a := range r.innerLoopSteps {
			if a.CanHandle(component) {
				log.Infof("### Invoking pipeline 'innerloop', action '%s' on %s", a.Name(), component.Name)
				if err := a.Handle(component, &r.client); err != nil {
					log.Error(err)
					return reconcile.Result{}, err
				}
			}
		}
	}

	// Check if the component is a Service to be installed from the catalog
	if component.Spec.Services != nil {
		for _, a := range r.serviceCatalogSteps {
			if a.CanHandle(component) {
				log.Infof("### Invoking'service catalog', action '%s' on %s", a.Name(), o.Name)
				if err := a.Handle(component, &r.client); err != nil {
					log.Error(err)
					return reconcile.Result{}, err
				}
			}
		}
	}

	// Check if the component is a Link
	if component.Spec.Link != nil {
		for _, a := range r.linkSteps {
			if a.CanHandle(component) {
				log.Infof("### Invoking'link', action '%s' on %s", a.Name(), component.Name)
				if err := a.Handle(component, &r.client); err != nil {
					log.Error(err)
					return reconcile.Result{}, err
				}
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

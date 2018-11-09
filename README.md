## Component's kubernetes operator

### How to play with operator locally

- Log on to an OpenShift cluster >=3.10 with cluster-admin rights
- Create a namespace `component-operator`
  ```bash
  $ oc new-project component-operator
  ```

- Deploy the resources : service account, rbac amd crd definition
  ```bash
  $ oc create -f deploy/sa.yaml
  $ oc create -f deploy/rbac.yaml
  $ oc create -f deploy/component/crd.yaml
  $ oc create -f deploy/export/crd.yaml
  ```

- Start the Operator locally
  ```bash
  $ oc new-project my-spring-app
  $ OPERATOR_NAME=component-operator WATCH_NAMESPACE=my-spring-app KUBERNETES_CONFIG=$HOME/.kube/config go run cmd/component-operator/main.go
  
- In a separate terminal a component's yaml file with a project `/path/to/project`
  ```bash
  $ echo component.yml << EOF
  apiVersion: component.k8s.io/v1alpha1
  kind: Component
  metadata:
    name: my-spring-boot
  spec:
    deployment: innerloop
    EOF
  $ oc apply -f component.yml 
  ```  

- Check if the operation has configured the `innerloop` with the following resources
  ```bash
  oc get all,pvc,component
  NAME                         READY     STATUS    RESTARTS   AGE
  pod/my-spring-boot-1-hrzcv   1/1       Running   0          11s
  
  NAME                                     DESIRED   CURRENT   READY     AGE
  replicationcontroller/my-spring-boot-1   1         1         1         12s
  
  NAME                         TYPE        CLUSTER-IP       EXTERNAL-IP   PORT(S)     AGE
  service/component-operator   ClusterIP   172.30.54.114    <none>        60000/TCP   3m
  service/my-spring-boot       ClusterIP   172.30.189.247   <none>        8080/TCP    15s
  
  NAME                                                REVISION   DESIRED   CURRENT   TRIGGERED BY
  deploymentconfig.apps.openshift.io/my-spring-boot   1          1         1         image(copy-supervisord:latest),image(dev-runtime:latest)
  
  NAME                                              DOCKER REPO                                      TAGS      UPDATED
  imagestream.image.openshift.io/copy-supervisord   172.30.1.1:5000/my-spring-app/copy-supervisord   latest    13 seconds ago
  imagestream.image.openshift.io/dev-runtime            172.30.1.1:5000/my-spring-app/dev-runtime            latest    12 seconds ago
  
  NAME                                      HOST/PORT                                           PATH      SERVICES         PORT      TERMINATION   WILDCARD
  route.route.openshift.io/my-spring-boot   my-spring-boot-my-spring-app.192.168.99.50.nip.io             my-spring-boot   <all>                   None
  
  NAME                            STATUS    VOLUME    CAPACITY   ACCESS MODES   STORAGECLASS   AGE
  persistentvolumeclaim/m2-data   Bound     pv0065    100Gi      RWO,ROX,RWX                   15s
  
  NAME                                        AGE
  component.component.k8s.io/my-spring-boot   15s
  ```
  
- Next scaffold (optional) a Spring Boot project, package it and push it to the dev's pod 

  ```bash
  cd /path/to/project
  sd create -t rest -i my-spring-boot
  mvn clean package
  sd push --mode binary
  sd exec start
  URL="http://$(oc get routes/my-spring-boot -o jsonpath='{.spec.host}')"
  curl $URL/api/greeting
  ```

- To cleanup the project installed (component)
  ```bash  
  $ oc delete components,route,svc,is,pvc,dc --all=true && 
  ```
  
** REMARK ** : You can also use the go client to create and publish a component CRD using the API

  ```bash
  $ go run cmd/sd/sd.go create my-spring-boot
  ```  
  
### How to install the operator on the cluster

  ```bash
  operator-sdk build quay.io/snowdrop/component-operator
  docker push quay.io/snowdrop/component-operator
  oc create -f deploy/operator.yaml
  ```  

### Cleanup

  ```bash
  oc delete -f deploy/cr.yaml
  oc delete -f deploy/crd.yaml
  oc delete -f deploy/operator.yaml
  oc delete -f deploy/rbac.yaml
  oc delete -f deploy/sa.yaml
  ```    

### How To create the operator, crd

Instructions followed to create the Component's CRD, operator using the `operator-sdk`'s kit

- Execute this command within the `$GOPATH/github.com/$ORG/` folder is a terminal
  ```bash
  operator-sdk new component-operator --api-version=component.k8s.io/v1alpha1 --kind=Component --skip-git-init
  operator-sdk add api --api-version=component.k8s.io/v1alpha1 --kind=Component 
  ```
  using the following parameters 

  Name of the folder to be created : `component-operator`
  Api Group Name   : `component.k8s.io`
  Api Version      : `v1alpha1`
  Kind of Resource : `Component` 

- Build and push the `component-operator` image to `quai.io`s registry
  ```bash
  $ operator-sdk build quay.io/snowdrop/component-operator
  $ docker push quay.io/snowdrop/component-operator
  ```
  
- Update the operator's manifest to use the built image name
  ```bash
  sed -i 's|REPLACE_IMAGE|quay.io/snowdrop/component-operator|g' deploy/operator.yaml
  ```
- Create a namespace `component-operator`

- Deploy the component-operator
  ```bash
  oc create -f deploy/sa.yaml
  oc create -f deploy/rbac.yaml
  oc create -f deploy/crd.yaml
  oc create -f deploy/operator.yaml
  ```

- By default, creating a custom resource triggers the `component-operator` to deploy a busybox pod
  ```bash
  oc create -f deploy/cr.yaml
  ```

- Verify that the busybox pod is created
  ```bash
  oc get pod -l app=busy-box
  NAME            READY     STATUS    RESTARTS   AGE
  busy-box   1/1       Running   0          50s
  ```

- Cleanup
  ```bash
  oc delete -f deploy/cr.yaml
  oc delete -f deploy/crd.yaml
  oc delete -f deploy/operator.yaml
  oc delete -f deploy/rbac.yaml
  oc delete -f deploy/sa.yaml
  ```

- Start operator locally

  ```bash
  $ oc new-project my-spring-app
  $ OPERATOR_NAME=component-operator WATCH_NAMESPACE=my-spring-app KUBERNETES_CONFIG=/Users/dabou/.kube/config go run cmd/component-operator/main.go
  
  $ oc delete components,route,svc,is,pvc,dc --all=true && go run cmd/sd/sd.go create my-spring-boot
  OR
  $ oc apply -f deploy/component1.yml
  $ oc get all,pvc,components,dc
  
  oc delete components,route,svc,is,pvc,dc --all=true
  ```  

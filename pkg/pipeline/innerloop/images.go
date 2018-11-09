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
	"github.com/snowdrop/component-operator/pkg/apis/component/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	zero          = int64(0)
	deleteOptions = &metav1.DeleteOptions{GracePeriodSeconds: &zero}
	image         = make(map[string]string)
)

func init() {
	image["java"] = "quay.io/snowdrop/spring-boot-s2i"
	image["nodejs"] = "docker.io/centos/nodejs-8-centos7"
	image["supervisord"] = "quay.io/snowdrop/supervisord"
}

func CreateTypeImage(dockerImage bool, name string, tag string, repo string, annotationCmd bool) v1alpha1.Image {
	return v1alpha1.Image{
		DockerImage:    dockerImage,
		Name:           name,
		Repo:           repo,
		AnnotationCmds: annotationCmd,
		Tag:            tag,
	}
}

func GetSupervisordImage() []v1alpha1.Image {
	return []v1alpha1.Image{
		CreateTypeImage(true, "copy-supervisord", "latest", image["supervisord"], true),
	}
}

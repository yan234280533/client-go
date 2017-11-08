/*
Copyright 2017 The Kubernetes Authors.

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

// Note: the example only works with the code within the same release/branch.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	//appsv1beta1 "k8s.io/api/apps/v1beta1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/client-go/util/retry"
	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/printers"
	"k8s.io/apimachinery/pkg/util/intstr"

	"bytes"
	"errors"
)

func PrintYamlObj(obj runtime.Object) (string,error) {
	if ( nil == obj ) {
		return "", errors.New("paramter obj is nil")
	}

	var print printers.ResourcePrinter = &printers.YAMLPrinter{}

	var buf bytes.Buffer
	err :=print.PrintObj(obj,&buf)

	if err == nil {
		return buf.String(), nil
	} else {
		return "", err
	}
}

func main() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	serviceClient := clientset.Core().Services(apiv1.NamespaceDefault)

	servicePort := apiv1.ServicePort{
			Protocol: "TCP",
			Port:     int32(80),
			NodePort: int32(0),
			Name:     "tst",
			TargetPort: intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: int32(80),
			},
		}

	service := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "nginx",
		},
		Spec: apiv1.ServiceSpec{
			Ports:[]apiv1.ServicePort{servicePort},
		},
	}

	// Create Service
	fmt.Println("Creating service...")
	result, err := serviceClient.Create(service)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Created service %q.\n", result.GetObjectMeta().GetName())

	result.Kind = "Service"
	result.APIVersion = "v1"

	textStr,err := PrintYamlObj(result)
		if err != nil {
		panic(err)
	}

	fmt.Printf("Convert service:\n%s", textStr)

	// Update Deployment
	prompt()
	fmt.Println("Updating service...")
	//    You have two options to Update() this Deployment:
	//
	//    1. Modify the "deployment" variable and call: Update(deployment).
	//       This works like the "kubectl replace" command and it overwrites/loses changes
	//       made by other clients between you Create() and Update() the object.
	//    2. Modify the "result" returned by Get() and retry Update(result) until
	//       you no longer get a conflict error. This way, you can preserve changes made
	//       by other clients between Create() and Update(). This is implemented below
	//			 using the retry utility package included with client-go. (RECOMMENDED)
	//
	// More Info:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#concurrency-control-and-consistency

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Retrieve the latest version of Deployment before attempting update
		// RetryOnConflict uses exponential backoff to avoid exhausting the apiserver
		result, getErr := serviceClient.Get("nginx", metav1.GetOptions{})
		if getErr != nil {
			panic(fmt.Errorf("Failed to get latest version of service: %v", getErr))
		}

		result.Spec.Ports[0].Port = 81
		_, updateErr :=serviceClient.Update(result)
		return updateErr
	})
	if retryErr != nil {
		panic(fmt.Errorf("Update failed: %v", retryErr))
	}
	fmt.Println("Updated service...")

	// List service
	prompt()
	fmt.Printf("Listing service in namespace %q:\n", apiv1.NamespaceDefault)
	list, err := serviceClient.List(metav1.ListOptions{})
	if err != nil {
		panic(err)
	}
	for _, d := range list.Items {
		fmt.Printf(" * %s (%s clusterIP)\n", d.Name, d.Spec.ClusterIP)
	}

	// Delete service
	prompt()
	fmt.Println("Deleting service...")
	deletePolicy := metav1.DeletePropagationForeground
	if err := serviceClient.Delete("nginx", &metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}); err != nil {
		panic(err)
	}
	fmt.Println("Deleted service.")
}

func prompt() {
	fmt.Printf("-> Press Return key to continue.")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		break
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	fmt.Println()
}

func int32Ptr(i int32) *int32 { return &i }

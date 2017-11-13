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
	"fmt"
	"os"
	"time"
	"io"
	"io/ioutil"
	"strings"
	goerrors "errors"

	"k8s.io/apimachinery/pkg/runtime"
	v1beta1 "k8s.io/api/apps/v1beta1"
	apiv1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/kubernetes/pkg/apis/extensions"

	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/client-go/tools/clientcmd"
	restclient "k8s.io/client-go/rest"
	"k8s.io/kubernetes/pkg/kubectl/resource"

	"k8s.io/client-go/util/homedir"
	"path/filepath"
	"flag"
)

type Result []*resource.Info

// Client represents a client capable of communicating with the Kubernetes API.
type Client struct {
	cmdutil.Factory
	// IncludeThirdPartyAPIs indicates whether to load "dynamic" APIs.
	//
	// This requires additional calls to the Kubernetes API server, and these calls
	// are not supported by all versions. Additionally, during testing, initializing
	// a client will still attempt to contact a live server. In these situations,
	// this flag may need to be disabled.
	IncludeThirdPartyAPIs bool

	// SchemaCacheDir is the path for loading cached schema.
	SchemaCacheDir string
}

// New create a new Client
func NewClient(config clientcmd.ClientConfig) *Client {
	return &Client{
		Factory:               cmdutil.NewFactory(config),
		IncludeThirdPartyAPIs: true,
		SchemaCacheDir: clientcmd.RecommendedSchemaFile,
	}
}

// scrubValidationError removes kubectl info from the message
func scrubValidationError(err error) error {
	if err == nil {
		return nil
	}
	const stopValidateMessage = "if you choose to ignore these errors, turn validation off with --validate=false"

	if strings.Contains(err.Error(), stopValidateMessage) {
		return goerrors.New(strings.Replace(err.Error(), "; "+stopValidateMessage, "", -1))
	}
	return err
}

func (c *Client) newBuilder(namespace string, reader io.Reader) *resource.Builder {
	return c.NewBuilder().
		ContinueOnError().
		NamespaceParam(namespace).
		DefaultNamespace().
		Stream(reader, "").
		Flatten()
}

// Build validates for Kubernetes objects and returns resource Infos from a io.Reader.
func (c *Client) Build(namespace string, reader io.Reader) (Result, error) {
	var result Result

	schema, err := c.Validator(true)
	if err != nil {
		fmt.Printf("warning: failed to load schema: %s\n", err)
	}
	result, err = c.NewBuilder().
		ContinueOnError().
		Schema(schema).
		NamespaceParam(namespace).
		DefaultNamespace().
		Stream(reader, "").
		Flatten().
		Do().Infos()

	return result, scrubValidationError(err)
}

func CreateClientConfig(restConfig *restclient.Config,) (clientcmd.ClientConfig, error) {

	overrides := &clientcmd.ConfigOverrides{}

	if restConfig.Insecure == true {
		overrides.ClusterDefaults.CertificateAuthorityData = restConfig.CAData
		overrides.ClusterDefaults.Server = restConfig.Host

	} else {
		overrides.ClusterDefaults.Server = restConfig.Host
	}

	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(), overrides)

	return clientConfig, nil
}

func PrintObjectType(obj runtime.Object) (error){

	fmt.Printf("Kind  %T\n",obj)

	switch typed := obj.(type) {
	case *v1beta1.Deployment:
		fmt.Printf("Kind is appsv1beta1.Deployment\n")
		fmt.Printf("typed:%s",typed.Kind)
		return nil
	case *extensionsv1beta1.Deployment:
		fmt.Printf("Kind is extensionsv1beta1.Deployment\n")
		fmt.Printf("typed:%s", typed.Kind)
		return nil
	case *extensions.Deployment:
		fmt.Printf("Kind is extensions.Deployment\n")
		fmt.Printf("typed:%s", typed.Kind)
		return nil
	default:
		fmt.Printf("Unsupported kind when set object type: %v\n", obj)
		return fmt.Errorf("Unsupported kind when set object type\n")
	}
	return nil
}

func read3(path string) string {
    fi, err := os.Open(path)
    if err != nil {
        panic(err)
    }
    defer fi.Close()
    fd, err := ioutil.ReadAll(fi)
    return string(fd)
}


func main() {

	file := "create-update-delete-deployment.yaml"

	start := time.Now()
	str := read3(file)
	t3 := time.Now()
	fmt.Printf("Cost time %v\n", t3.Sub(start))
	fmt.Printf("str:%s\n", str)
	fmt.Printf("------------------------------------------------\n")

	reader := strings.NewReader(str)

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

	fmt.Printf("config %#v \n",config)

	clientconfig, err := CreateClientConfig(config)
	if err != nil {
		panic(err)
	}

	fmt.Printf("clientconfig %#v \n",clientconfig)

	client := NewClient(clientconfig)
	kube_result,err :=  client.Build(apiv1.NamespaceDefault,reader)

	var result []*resource.Info
	for _, value := range kube_result {
		//fmt.Printf("range is  %+v\n",value)
		result = append(result, value)
		fmt.Printf("value %#v \n",value)
		PrintObjectType(value.Object)
		PrintObjectType(value.VersionedObject)
	}

	for _, value := range result {
		fmt.Printf("Build is over, kind: %#v\n", value.Object.GetObjectKind())
	}

        return
}

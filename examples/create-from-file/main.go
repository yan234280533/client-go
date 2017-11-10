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

	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	v1beta1 "k8s.io/api/apps/v1beta1"
)

func PrintObjectType(obj runtime.Object) (error){
	switch typed := obj.(type) {
	case *v1beta1.Deployment:
		fmt.Printf("Kind is v1beta1.Deployment\n")
		fmt.Printf("typed:%s",typed.Kind)
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
	d := yaml.NewYAMLOrJSONDecoder(reader, 4096)

        for {
		data := unstructured.Unstructured{}
		if err := d.Decode(&data); err != nil {
			if err == io.EOF {
				fmt.Printf("decode is over\n")
				break
			}
			fmt.Printf("decode err:%s\n",err.Error())
			break
		}

		version := data.GetAPIVersion()
		kind := data.GetKind()

		fmt.Printf("version:%s, kind: %s\n",version,kind)

		b, err := data.MarshalJSON()
		if err != nil {
			fmt.Printf("MarshalJSON is failed")
			break
		}

		obj, _, err := unstructured.UnstructuredJSONScheme.Decode(b, nil, nil)
		if err != nil {
			fmt.Printf("Decode is failed")
			break
		}

		err = PrintObjectType(obj)
		if err != nil {
			fmt.Printf("PrintObjectType is failed")
		}
	}
}

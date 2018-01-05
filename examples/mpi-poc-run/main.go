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
	"flag"
	"fmt"
	"path/filepath"
	"strings"
	"net"
	yaml "github.com/davidje13/yaml.v2"
	"errors"

	//appsv1beta1 "k8s.io/api/apps/v1beta1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	//extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	//appsv1beta1 "k8s.io/client-go/pkg/apis/apps/v1beta1"
	extensionsv1beta1 "k8s.io/client-go/pkg/apis/extensions/v1beta1"

	//"k8s.io/apimachinery/pkg/labels"
	//"k8s.io/apimachinery/pkg/fields"
	"time"
)

const (
	LABEL_APP = "qcloud-app"
)

const (
	VASP_SERVICE_NAME = "vasp"
	VASP_IMAGE = "ccr.ccs.tencentyun.com/xtalpi/vasp:std-ssh"
	VASP_REPLICAS = 2
	VASP_MASTER_SERVICE_NAME = "vasp-master"
	VASP_MASTER_IMAGE = "ccr.ccs.tencentyun.com/xtalpi/vasp:std"
	VASP_CONFIGMAP_NAME = "vasp-config"
	VASP_HOSTFILE_NAME = "hostsfile"
	VASP_HOSTFILE_VOL_NAME = "hostsfilevol"

)

/*func generateApiListOptRegex(labelMap map[string]string, fieldMap map[string]string, labelsRegex string) (apiv1.ListOptions, error) {
	labelSelector := labels.Everything()
	fieldSelector := fields.Everything()
	var opt apiv1.ListOptions
	var err error

	if len(labelMap) > 0 {
		labelSelector = labels.SelectorFromSet(labelMap)
	}

	if len(fieldMap) > 0 {
		fieldSelector = fields.SelectorFromSet(fieldMap)
	}

	if labelsRegex != "" {
		labelSelector, err = labels.Parse(labelsRegex)
		if err != nil {
			return opt, err
		}
	}

	opt = apiv1.ListOptions{
		LabelSelector: labelSelector,
		FieldSelector: fieldSelector,
	}
	return opt, nil
}*/

func MarshalConfigMapData(configValueMap map[string]string)  (string, error)  {
	var strYaml string
	var err error = nil

	if len(configValueMap) == 0 {
		return strYaml,nil
	}

	strByte, err := yaml.Marshal(configValueMap)
	if err != nil {
		fmt.Errorf("Marshal the strMap failed, error=%s.", err.Error())
		return strYaml, errors.New(fmt.Sprintf("Marshal the strMap failed, error=%s.", err.Error()))
	}

	strYaml = string(strByte)
	return strYaml,nil
}

func getLocalIp()(string,error) {
	var ipAddr = "localhost"

	addrSlice, err := net.InterfaceAddrs()
	if nil != err {
		fmt.Errorf("Get local IP addr failed,%s",err.Error())
		return ipAddr,errors.New(fmt.Sprintf("Get local IP addr failed,%s",err.Error()))
	}
	for _, addr := range addrSlice {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if nil != ipnet.IP.To4() {
				ipAddr = ipnet.IP.String()
				return ipAddr,nil
			}
		}
	}
	return ipAddr,errors.New(fmt.Sprintf("Not found local IP addr"))
}

func int32Ptr(i int32) *int32 { return &i }

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

	namespace := "a1"

	//创建vasp的服务
	deploymentsClient := clientset.ExtensionsV1beta1().Deployments(namespace)
	deployLabels := map[string]string{LABEL_APP: VASP_SERVICE_NAME,}
	imageSecret    := apiv1.LocalObjectReference{Name:"qcloudregistrykey"}

	deployment := &extensionsv1beta1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: VASP_SERVICE_NAME,
			Labels: deployLabels,
		},
		Spec: extensionsv1beta1.DeploymentSpec{
			Replicas: int32Ptr(VASP_REPLICAS),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{LABEL_APP: VASP_SERVICE_NAME},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:deployLabels,
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:  VASP_SERVICE_NAME,
							Image: VASP_IMAGE,
							Ports: []apiv1.ContainerPort{
								{
									Name:          "sshd",
									Protocol:      apiv1.ProtocolTCP,
									ContainerPort: 22,
								},
							},
						},
					},
					ImagePullSecrets:[]apiv1.LocalObjectReference{imageSecret},
				},
			},
		},
	}

	// Create Vasp Deployment
	fmt.Printf("Creating %s deployment...\n",VASP_SERVICE_NAME)
	_, err = deploymentsClient.Create(deployment)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Creat deployment %s succeed...\n",VASP_SERVICE_NAME)

	//获取Pod列表，获取Pod 的IP
	podClient := clientset.Core().Pods(namespace)

	//strRegex := fmt.Sprintf("%s in (%s)",LABEL_APP,VASP_SERVICE_NAME)
	//fmt.Println("regex: %s",strRegex)

	/*opt, err := generateApiListOptRegex(nil, nil, strRegex)
	if err != nil {
		panic(err)
	}*/
	opt := metav1.ListOptions{}

	var ipList []string
	var allRuning bool = false
	for j := 0; j<100 ; j++ {
		time.Sleep(60)

		fmt.Printf("list pod loop(%d) \n",j)

		result, err := podClient.List(opt)
		if err != nil {
			panic(err)
		}

		ipList = []string{}
		for i := 0; i < len(result.Items); i++ {
			if (result.Items[i].Status.Phase != apiv1.PodRunning) || (result.Items[i].Status.PodIP == "") {
                                allRuning = false
				break;
			}else{
				allRuning = true
				ipList = append(ipList,result.Items[i].Status.PodIP)
			}
		}

		fmt.Printf("allRuning is %v, num %d \n",allRuning,len(ipList))

		if allRuning {
			if len(ipList) == VASP_REPLICAS {
				break;
			}
		}
	}

	fmt.Printf("list pod succeed,%s\n, len(%d)",strings.Join(ipList,","),len(ipList))

	//将获取IP转换成对应的configmap
	configmap := apiv1.ConfigMap{}
	configmap.Name = VASP_CONFIGMAP_NAME
	configmap.Namespace = namespace

	labels := make(map[string]string, 0)
	labels[LABEL_APP] = VASP_MASTER_SERVICE_NAME
	configmap.Labels = labels

	dateMaps := make(map[string]string, 0)
	dateMaps[VASP_HOSTFILE_NAME] = strings.Join(ipList,"\n")
	configmap.Data = dateMaps

	//创建vasp-master服务对应的configmap
	configClient := clientset.Core().ConfigMaps(namespace)
	_, err = configClient.Create(&configmap)
	if err != nil {
		panic(err)
	}

	fmt.Printf("create configmap %s succeed\n",VASP_CONFIGMAP_NAME)

	//创建vasp-master的服务，启动执行任务
	deployMasterLabels := map[string]string{LABEL_APP: VASP_SERVICE_NAME,}
	ipAddr, err := getLocalIp()
	if err != nil {
		panic(err)
	}

	fmt.Printf("get addr ip  succeed,%s\n",ipAddr)

	var volume apiv1.Volume
	volume.Name = VASP_HOSTFILE_VOL_NAME

	var mode int32
	mode = 0777

	volume.VolumeSource = apiv1.VolumeSource{
		ConfigMap: &apiv1.ConfigMapVolumeSource{
			LocalObjectReference: apiv1.LocalObjectReference{
				Name: VASP_CONFIGMAP_NAME,
			},
			Items: []apiv1.KeyToPath{
				{
					Key:  VASP_HOSTFILE_NAME,
					Path: VASP_HOSTFILE_NAME,
					Mode: &mode,
				},
			},
		},

	}

	var volumeMount = apiv1.VolumeMount{
		Name: VASP_HOSTFILE_VOL_NAME,
		ReadOnly: false,
		MountPath: fmt.Sprintf("/mnt/%s",VASP_HOSTFILE_NAME),
                SubPath: VASP_HOSTFILE_NAME,
	}

	deploymentMaster := &extensionsv1beta1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: VASP_MASTER_SERVICE_NAME,
			Labels: deployMasterLabels,
		},
		Spec: extensionsv1beta1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: deployMasterLabels,
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:deployMasterLabels,
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:  VASP_MASTER_SERVICE_NAME,
							Image: VASP_MASTER_IMAGE,
							Command: []string{"sleep"},
                                                        Args: []string{"36000"},
							VolumeMounts: []apiv1.VolumeMount{volumeMount},
						},
					},
					Volumes:[]apiv1.Volume{volume},
					ImagePullSecrets:[]apiv1.LocalObjectReference{imageSecret},
				},
			},
		},
	}

	// Create Vasp Master Deployment
	fmt.Printf("Creating %s deployment...\n",VASP_MASTER_SERVICE_NAME)
	_, err = deploymentsClient.Create(deploymentMaster)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Creat deployment %s succeed...\n",VASP_MASTER_SERVICE_NAME)

	return
}

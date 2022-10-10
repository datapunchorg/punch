/*
Copyright 2022 DataPunch Organization

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kubelib

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/datapunchorg/punch/pkg/common"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type KubeConfig struct {
	ConfigFile string
	ApiServer  string
	CAFile     string
	CA         []byte
	KubeToken  string
}

func (t *KubeConfig) Cleanup() error {
	if t.CAFile == "" {
		return nil
	}
	err := os.Remove(t.CAFile)
	if err != nil {
		return fmt.Errorf("failed to delete CA file %s: %s", t.CAFile, err.Error())
	}

	t.CA = nil
	t.KubeToken = ""

	return nil
}

func CheckPodsInPhases(clientset *kubernetes.Clientset, namespace string, podNamePrefix string, podPhases []v1.PodPhase) (bool, error) {
	podList, err := clientset.CoreV1().Pods(namespace).List(
		context.TODO(),
		metav1.ListOptions{})
	if err != nil {
		return false, err
	}
	podsReady := true
	podsCount := 0
	for _, pod := range podList.Items {
		if !strings.HasPrefix(pod.Name, podNamePrefix) {
			continue
		}
		podsCount++
		if !containsPodPhase(podPhases, pod.Status.Phase) {
			log.Printf("Pod %s in namespace %s is in phase %s (not expected: %s)", pod.Name, namespace, pod.Status.Phase, podPhases)
			podsReady = false
			break
		}
	}
	if podsCount == 0 {
		return false, nil
	} else {
		return podsReady, nil
	}
}

func CheckPodsDeleted(clientset *kubernetes.Clientset, namespace string, podNamePrefix string) (bool, error) {
	podList, err := clientset.CoreV1().Pods(namespace).List(
		context.TODO(),
		metav1.ListOptions{})
	if err != nil {
		return false, err
	}
	podsCount := 0
	for _, pod := range podList.Items {
		if !strings.HasPrefix(pod.Name, podNamePrefix) {
			continue
		}
		podsCount++
	}
	if podsCount == 0 {
		return true, nil
	} else {
		return false, nil
	}
}

func WaitPodsInPhases(clientset *kubernetes.Clientset, namespace string, podNamePrefix string, podPhases []v1.PodPhase) error {
	return common.RetryUntilTrue(func() (bool, error) {
		log.Printf("Checking whether pod %s*** in namespace %s is in phase %s", podNamePrefix, namespace, podPhases)
		result, err := CheckPodsInPhases(clientset, namespace, podNamePrefix, podPhases)
		if err != nil {
			return result, err
		}
		return result, err
	},
		10*time.Minute,
		10*time.Second)
}

func WaitPodsDeleted(clientset *kubernetes.Clientset, namespace string, podNamePrefix string) error {
	return common.RetryUntilTrue(func() (bool, error) {
		log.Printf("Checking whether pod %s*** in namespace %s deleted", podNamePrefix, namespace)
		result, err := CheckPodsDeleted(clientset, namespace, podNamePrefix)
		if err != nil {
			return result, err
		}
		return result, err
	},
		10*time.Minute,
		10*time.Second)
}

func GetServiceLoadBalancerHostPorts(clientset *kubernetes.Clientset, namespace string, serviceName string) ([]common.HostPort, error) {
	service, err := clientset.CoreV1().Services(namespace).Get(
		context.TODO(),
		serviceName,
		metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	var hostPorts []common.HostPort

	switch service.Spec.Type {
	case v1.ServiceTypeNodePort:
		for _, port := range service.Spec.Ports {
			hostPort := common.HostPort{
				Host: "localhost",
				Port: port.NodePort,
			}
			hostPorts = append(hostPorts, hostPort)
		}
	case v1.ServiceTypeLoadBalancer:
		for _, ingress := range service.Status.LoadBalancer.Ingress {
			if ingress.Hostname != "" {
				for _, port := range service.Spec.Ports {
					hostPort := common.HostPort{
						Host: ingress.Hostname,
						Port: port.Port,
					}
					hostPorts = append(hostPorts, hostPort)
				}
			}
		}
	default:
		return nil, fmt.Errorf("invalid service spec type: %v", service.Spec.Type)
	}

	return hostPorts, nil
}

func GetKubeConfigPath() string {
	var kubeConfigEnv = os.Getenv("KUBECONFIG")
	if len(kubeConfigEnv) == 0 {
		return os.Getenv("HOME") + "/.kube/config"
	}
	return kubeConfigEnv
}

func containsPodPhase(slice []v1.PodPhase, value v1.PodPhase) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

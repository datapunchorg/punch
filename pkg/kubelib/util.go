/*
Copyright 2022 DataPunch Project

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
	"github.com/datapunchorg/punch/pkg/common"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"log"
	"os"
	"strings"
	"time"
)

type KubeConfig struct {
	ConfigFile string
	ApiServer string
	CAFile string
	CA     []byte
	KubeToken string
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

func AppendHelmKubeArguments(arguments []string, kubeConfig KubeConfig) []string {
	if kubeConfig.ConfigFile != "" {
		arguments = append(arguments, "--kubeconfig", kubeConfig.ConfigFile)
	}
	if kubeConfig.ApiServer != "" {
		arguments = append(arguments, "--kube-apiserver", kubeConfig.ApiServer)
	}
	if kubeConfig.CAFile != "" {
		arguments = append(arguments, "--kube-ca-file", kubeConfig.CAFile)
	}
	if kubeConfig.KubeToken != "" {
		arguments = append(arguments, "--kube-token", kubeConfig.KubeToken)
	}
	return arguments
}

func CheckPodsInPhase(clientset *kubernetes.Clientset, namespace string, podNamePrefix string, podPhase v1.PodPhase) (bool, error) {
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
		if pod.Status.Phase != podPhase {
			log.Printf("Pod %s in namespace %s is in phase %s (not expected: %s)", pod.Name, namespace, pod.Status.Phase, podPhase)
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

func WaitPodsInPhase(clientset *kubernetes.Clientset, namespace string, podNamePrefix string, podPhase v1.PodPhase) error {
	return common.RetryUntilTrue(func() (bool, error) {
			log.Printf("Checking whether pod %s* in namespace %s is in phase %s", podNamePrefix, namespace, podPhase)
			result, err := CheckPodsInPhase(clientset, namespace, podNamePrefix, podPhase)
			if err != nil {
				return result, err
			}
			return result, err
		},
		10 * time.Minute,
		10 * time.Second)
}

func GetServiceLoadBalancerUrls(clientset *kubernetes.Clientset, namespace string, serviceName string) ([]string, error) {
	service, err := clientset.CoreV1().Services(namespace).Get(
		context.TODO(),
		serviceName,
		metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	var ingressHostNames []string

	for _, ingress := range service.Status.LoadBalancer.Ingress {
		if ingress.Hostname != "" {
			ingressHostNames = append(ingressHostNames, ingress.Hostname)
		}
	}

	return ingressHostNames, nil
}

func GetKubeConfigPath() string {
	var kubeConfigEnv = os.Getenv("KUBECONFIG")
	if len(kubeConfigEnv) == 0 {
		return os.Getenv("HOME") + "/.kube/config"
	}
	return kubeConfigEnv
}

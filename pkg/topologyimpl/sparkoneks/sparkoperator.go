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

package sparkoneks

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/datapunchorg/punch/pkg/awslib"
	"github.com/datapunchorg/punch/pkg/common"
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/kubelib"
	v1 "k8s.io/api/core/v1"
	v13 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

func DeploySparkOperator(commandEnvironment framework.CommandEnvironment, sparkComponentSpec SparkComponentSpec, region string, eksClusterName string, s3Bucket string) error {
	operatorNamespace := sparkComponentSpec.Operator.Namespace

	eventLogDir := sparkComponentSpec.Gateway.SparkEventLogDir
	if !commandEnvironment.GetBoolOrElse(framework.CmdEnvWithMinikube, false) {
		if strings.HasPrefix(strings.ToLower(eventLogDir), "s3") {
			dummyFileUrl := eventLogDir
			if !strings.HasSuffix(dummyFileUrl, "/") {
				dummyFileUrl += "/"
			}
			dummyFileUrl += "dummy.txt"
			log.Printf("Uploading dummy file %s to Spark event log directory %s to make sure the directry exits (Spark application and history server will fail if the directory does not exist)", dummyFileUrl, eventLogDir)
			err := awslib.UploadDataToS3Url(region, dummyFileUrl, strings.NewReader(""))
			if err != nil {
				return fmt.Errorf("failed to create dummy file %s in Spark event log directory %s to make sure the directry exits (Spark application and history server will fail if the directory does not exist): %s", dummyFileUrl, eventLogDir, err.Error())
			}
		}
	}

	_, clientset, err := awslib.CreateKubernetesClient(region, commandEnvironment.Get(framework.CmdEnvKubeConfig), eksClusterName)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %s", err.Error())
	}

	sparkApplicationNamespace := "spark-01"

	namespace, err := clientset.CoreV1().Namespaces().Create(
		context.TODO(),
		&v1.Namespace{
			ObjectMeta: v12.ObjectMeta{
				Name: sparkApplicationNamespace,
			},
		},
		v12.CreateOptions{},
	)
	if err != nil {
		if !awslib.AlreadyExistsMessage(err.Error()) {
			return fmt.Errorf("failed to create Spark application namespace %s: %v", sparkApplicationNamespace, err)
		} else {
			log.Printf("Namespace %s already exists, do not create it again", sparkApplicationNamespace)
		}
	} else {
		log.Printf("Created Spark application namespace %s", namespace.Name)
	}

	err = InstallSparkOperatorHelm(commandEnvironment, sparkComponentSpec, region, eksClusterName, s3Bucket)
	if err != nil {
		return err
	}

	// CreateSparkServiceAccount(clientset, operatorNamespace, sparkApplicationNamespace, "spark")

	helmInstallName := sparkComponentSpec.Operator.HelmInstallName
	err = CreateApiGatewayService(clientset, operatorNamespace, helmInstallName, helmInstallName)
	if err != nil {
		return err
	}

	sparkOperatorPodNamePrefix := helmInstallName
	err = kubelib.WaitPodsInPhases(clientset, operatorNamespace, sparkOperatorPodNamePrefix, []v1.PodPhase{v1.PodRunning})
	if err != nil {
		return fmt.Errorf("pod %s*** in namespace %s is not in phase %s", sparkOperatorPodNamePrefix, operatorNamespace, v1.PodRunning)
	}

	// Retry and handle error like following
	// Failed to create ingress spark-operator-01 in namespace spark-operator-01 for service spark-operator-01: Internal error occurred: failed calling webhook "validate.nginx.ingress.kubernetes.io": Post "https://ingress-nginx-controller-admission.ingress-nginx.svc:443/networking/v1/ingresses?timeout=10s": context deadline exceeded
	return common.RetryUntilTrue(func() (bool, error) {
		err := CreateApiGatewayIngress(clientset, operatorNamespace, helmInstallName, helmInstallName)
		ignoreErrorMsg := "failed calling webhook \"validate.nginx.ingress.kubernetes.io\""
		if err != nil {
			if strings.Contains(err.Error(), ignoreErrorMsg) {
				log.Printf("Ignore error in creating ingress and will retry: %s", err.Error())
				return false, nil
			} else {
				return false, err
			}
		} else {
			return true, nil
		}
	},
		10*time.Minute,
		10*time.Second)
}

func CreateApiGatewayService(clientset *kubernetes.Clientset, namespace string, serviceName string, instanceName string) error {
	serviceType := "ClusterIP"
	_, err := clientset.CoreV1().Services(namespace).Create(
		context.TODO(),
		&v1.Service{
			ObjectMeta: v12.ObjectMeta{
				Namespace: namespace,
				Name:      serviceName,
				Annotations: map[string]string{
					// "service.beta.kubernetes.io/aws-load-balancer-healthcheck-protocol": "http",
					// "service.beta.kubernetes.io/aws-load-balancer-healthcheck-path": "/health",
				},
			},
			Spec: v1.ServiceSpec{
				Selector: map[string]string{
					"app.kubernetes.io/name":     "spark-operator",
					"app.kubernetes.io/instance": instanceName,
				},
				Ports: []v1.ServicePort{
					{
						Name:       "http",
						Protocol:   "TCP",
						Port:       80,
						TargetPort: intstr.FromInt(80),
					},
				},
				Type: v1.ServiceType(serviceType),
			},
		},
		v12.CreateOptions{})
	if err != nil {
		if !awslib.AlreadyExistsMessage(err.Error()) {
			return fmt.Errorf("failed to create API gateway service %s in namespace %s: %v", serviceName, namespace, err)
		} else {
			log.Printf("API gateway service %s in namespace %s already exists, do not create it again", serviceName, namespace)
		}
	} else {
		log.Printf("Created API gateway service %s in namespace %s", serviceName, namespace)
	}

	service, err := clientset.CoreV1().Services(namespace).Get(
		context.TODO(),
		serviceName,
		v12.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get API gateway service %s in namespace %s: %v", serviceName, namespace, err)
	}

	// TODO delete following?
	for _, ingress := range service.Status.LoadBalancer.Ingress {
		log.Printf("Got ingress %s for API gateway service %s in namespace %s", ingress.Hostname, serviceName, namespace)
	}

	return nil
}

func InstallSparkOperatorHelm(commandEnvironment framework.CommandEnvironment, sparkComponentSpec SparkComponentSpec, region string, eksClusterName string, s3Bucket string) error {
	// helm install my-release spark-operator/spark-operator --namespace spark-operator --create-namespace --set sparkJobNamespace=default

	kubeConfig, err := awslib.CreateKubeConfig(region, commandEnvironment.Get(framework.CmdEnvKubeConfig), eksClusterName)
	if err != nil {
		return fmt.Errorf("failed to get kube config: %s", err)
	}

	defer kubeConfig.Cleanup()

	installName := sparkComponentSpec.Operator.HelmInstallName
	operatorNamespace := sparkComponentSpec.Operator.Namespace
	sparkApplicationNamespace := sparkComponentSpec.Operator.SparkApplicationNamespace

	arguments := []string{
		"--set", fmt.Sprintf("sparkJobNamespace=%s", sparkApplicationNamespace),
		"--set", fmt.Sprintf("image.repository=%s", sparkComponentSpec.Operator.ImageRepository),
		"--set", fmt.Sprintf("image.tag=%s", sparkComponentSpec.Operator.ImageTag),
		"--set", "serviceAccounts.spark.create=true",
		"--set", "serviceAccounts.spark.name=spark",
		"--set", "spark.gateway.user=" + sparkComponentSpec.Gateway.User,
		"--set", "spark.gateway.password=" + sparkComponentSpec.Gateway.Password,
		"--set", "spark.gateway.s3Region=" + region,
		"--set", "spark.gateway.s3Bucket=" + s3Bucket,
		// "--set", "webhook.enable=true",
	}

	if !commandEnvironment.GetBoolOrElse(framework.CmdEnvWithMinikube, false) {
		arguments = append(arguments, "--set")
		arguments = append(arguments, "spark.gateway.sparkEventLogEnabled=true")
		arguments = append(arguments, "--set")
		arguments = append(arguments, "spark.gateway.sparkEventLogDir="+sparkComponentSpec.Gateway.SparkEventLogDir)
	}

	if sparkComponentSpec.Gateway.HiveMetastoreUris != "" {
		arguments = append(arguments, "--set")
		arguments = append(arguments, "spark.gateway.hiveMetastoreUris="+sparkComponentSpec.Gateway.HiveMetastoreUris)
	}
	if sparkComponentSpec.Gateway.SparkSqlWarehouseDir != "" {
		arguments = append(arguments, "--set")
		arguments = append(arguments, "spark.gateway.sparkSqlWarehouseDir="+sparkComponentSpec.Gateway.SparkSqlWarehouseDir)
	}

	err = kubelib.InstallHelm(commandEnvironment.Get(framework.CmdEnvHelmExecutable), commandEnvironment.Get(CmdEnvSparkOperatorHelmChart), kubeConfig, arguments, installName, operatorNamespace)
	return err
}

func CreateSparkServiceAccount(clientset *kubernetes.Clientset, sparkOperatorNamespace string, sparkApplicationNamespace string, sparkServiceAccountName string) error {
	_, err := clientset.CoreV1().ServiceAccounts(sparkApplicationNamespace).Create(
		context.TODO(),
		&v1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{Name: sparkServiceAccountName},
		},
		metav1.CreateOptions{})
	if err != nil {
		if !awslib.AlreadyExistsMessage(err.Error()) {
			return fmt.Errorf("failed to create Spark service account %s in namespace %s: %s", sparkServiceAccountName, sparkApplicationNamespace, err.Error())
		} else {
			log.Printf("Spark service account %s in namespace %s already exists, do not create it again", sparkServiceAccountName, sparkApplicationNamespace)
		}
	} else {
		log.Printf("Created Spark service account %s in namespace %s", sparkServiceAccountName, sparkApplicationNamespace)
	}

	roleName := fmt.Sprintf("%s-role", sparkServiceAccountName)
	_, err = clientset.RbacV1().Roles(sparkApplicationNamespace).Create(
		context.TODO(),
		&rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{Name: roleName},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"pods"},
					Verbs:     []string{"*"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"services"},
					Verbs:     []string{"*"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"configmaps"},
					Verbs:     []string{"*"},
				},
			},
		},
		metav1.CreateOptions{})
	if err != nil {
		if !awslib.AlreadyExistsMessage(err.Error()) {
			return fmt.Errorf("failed to create Spark service account role %s in namespace %s: %s", roleName, sparkApplicationNamespace, err.Error())
		} else {
			log.Printf("Spark service account role %s in namespace %s already exists, do not create it again", roleName, sparkApplicationNamespace)
		}
	} else {
		log.Printf("Created Spark service account role %s in namespace %s", roleName, sparkApplicationNamespace)
	}

	roleBindingName := fmt.Sprintf("%s-role-binding", sparkServiceAccountName)
	_, err = clientset.RbacV1().RoleBindings(sparkApplicationNamespace).Create(
		context.TODO(),
		&rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: roleBindingName},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      sparkServiceAccountName,
					Namespace: sparkApplicationNamespace,
				},
			},
			RoleRef: rbacv1.RoleRef{
				Kind:     "Role",
				Name:     roleName,
				APIGroup: "rbac.authorization.k8s.io",
			},
		},
		metav1.CreateOptions{})
	if err != nil {
		if !awslib.AlreadyExistsMessage(err.Error()) {
			return fmt.Errorf("failed to create Spark service account role binding %s in namespace %s: %s", roleBindingName, sparkApplicationNamespace, err.Error())
		} else {
			log.Printf("Spark service account role binding %s in namespace %s already exists, do not create it again", roleBindingName, sparkApplicationNamespace)
		}
	} else {
		log.Printf("Created Spark service account role binding %s in namespace %s", roleBindingName, sparkApplicationNamespace)
	}
	return nil
}

func CreateApiGatewayIngress(clientset *kubernetes.Clientset, namespace string, ingressName string, serviceName string) error {
	path := "/sparkapi/"
	pathType := v13.PathTypePrefix
	log.Printf("Creating ingress %s in namespace %s for sevice %s", ingressName, namespace, serviceName)
	_, err := clientset.NetworkingV1().Ingresses(namespace).Create(
		context.TODO(),
		&v13.Ingress{
			ObjectMeta: v12.ObjectMeta{
				Name:      ingressName,
				Namespace: namespace,
				Annotations: map[string]string{
					"nginx.ingress.kubernetes.io/ssl-redirect":       "false",
					"nginx.ingress.kubernetes.io/force-ssl-redirect": "false",
					"nginx.ingress.kubernetes.io/proxy-body-size":    "1g",
				},
			},
			Spec: v13.IngressSpec{
				IngressClassName: aws.String("nginx"),
				Rules: []v13.IngressRule{
					v13.IngressRule{
						IngressRuleValue: v13.IngressRuleValue{
							HTTP: &v13.HTTPIngressRuleValue{
								Paths: []v13.HTTPIngressPath{
									v13.HTTPIngressPath{
										Path:     path,
										PathType: &pathType,
										Backend: v13.IngressBackend{
											Service: &v13.IngressServiceBackend{
												Name: serviceName,
												Port: v13.ServiceBackendPort{
													Number: 80,
												},
											},
										},
									},
								},
							},
						},
					},
				},
				// TODO use command like following to create secret
				// kubectl create secret tls aks-ingress-tls \
				//--namespace ingress-basic \
				//--key aks-ingress-tls.key \
				//--cert aks-ingress-tls.crt
				TLS: []v13.IngressTLS{
					{
						Hosts:      []string{"*.amazonaws.com"},
						SecretName: "tls-secret-name",
					},
				},
			},
		},
		v12.CreateOptions{},
	)
	if err != nil {
		// Check error like: admission webhook "validate.nginx.ingress.kubernetes.io" denied the request: host "_" and path "/sparkapi/" is already defined in ingress spark-operator-01/my1-spark-operator-01
		pathAlreadyDefinedMsg := fmt.Sprintf("path \"%s\" is already defined in ingress", path)
		if !awslib.AlreadyExistsMessage(err.Error()) && !strings.Contains(err.Error(), pathAlreadyDefinedMsg) {
			return fmt.Errorf("failed to create ingress %s in namespace %s for sevice %s (%s)", ingressName, namespace, serviceName, err.Error())
		} else {
			log.Printf("Ingress %s in namespace %s for sevice %s already exists, do not create it again", ingressName, namespace, serviceName)
		}
	} else {
		log.Printf("Created ingress %s in namespace %s for sevice %s", ingressName, namespace, serviceName)
	}
	return nil
}

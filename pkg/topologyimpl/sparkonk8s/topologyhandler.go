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

package sparkonk8s

import (
	"bytes"
	"fmt"
	"github.com/datapunchorg/punch/pkg/awslib"
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/kubelib"
	"github.com/datapunchorg/punch/pkg/resource"
	"gopkg.in/yaml.v3"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"text/template"
)

const (
	sparkcliJavaExampleCommandFormat   = `./sparkcli --user %s --password %s --insecure --url https://%s/sparkapi/v1 submit --class org.apache.spark.examples.SparkPi --image ghcr.io/datapunchorg/spark:spark-3.2.1-1643336295 --spark-version 3.2 --driver-memory 512m --executor-memory 512m local:///opt/spark/examples/jars/spark-examples_2.12-3.2.1.jar`
	sparkcliPythonExampleCommandFormat = `./sparkcli --user %s --password %s --insecure --url https://%s/sparkapi/v1 submit --image ghcr.io/datapunchorg/spark:pyspark-3.2.1-1643336295 --spark-version 3.2 --driver-memory 512m --executor-memory 512m %s`
)

func init() {
	framework.DefaultTopologyHandlerManager.AddHandler(KindSparkTopology, &TopologyHandler{})
}

type TopologyHandler struct {
}

func (t *TopologyHandler) Generate() (framework.Topology, error) {
	topology := createSparkTopologyTemplate()
	return &topology, nil
}

func (t *TopologyHandler) Parse(yamlContent []byte) (framework.Topology, error) {
	result := CreateDefaultSparkTopology(DefaultNamePrefix, ToBeReplacedS3BucketName)
	err := yaml.Unmarshal(yamlContent, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML (%s): \n%s", err.Error(), string(yamlContent))
	}
	return &result, nil
}

func (t *TopologyHandler) Resolve(topology framework.Topology, data framework.TemplateData) (framework.Topology, error) {
	topologyBytes, err := yaml.Marshal(topology)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal topology: %s", err.Error())
	}
	yamlContent := string(topologyBytes)

	tmpl, err := template.New("").Parse(yamlContent) // .Option("missingkey=error")?
	if err != nil {
		return nil, fmt.Errorf("failed to parse topology template (%s): %s", err.Error(), yamlContent)
	}

	sparkData := CreateSparkTemplateData(data)

	buffer := bytes.Buffer{}
	err = tmpl.Execute(&buffer, &sparkData)
	if err != nil {
		return nil, fmt.Errorf("failed to execute topology template: %s", err.Error())
	}
	resolvedContent := buffer.String()
	resolvedTopology, err := t.Parse([]byte(resolvedContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse resolved topology (%s): %s", err.Error(), resolvedContent)
	}

	resolvedSparkTopology := resolvedTopology.(*SparkTopology)
	if resolvedSparkTopology.Spec.ApiGateway.UserPassword == "" || resolvedSparkTopology.Spec.ApiGateway.UserPassword == framework.TemplateNoValue {
		return nil, fmt.Errorf("spec.apiGateway.userPassword is emmpty, please provide the value for the password")
	}

	err = checkCmdEnvFolderExists(resolvedSparkTopology.Metadata, CmdEnvNginxHelmChart)
	if err != nil {
		return nil, err
	}

	err = checkCmdEnvFolderExists(resolvedSparkTopology.Metadata, CmdEnvSparkOperatorHelmChart)
	if err != nil {
		return nil, err
	}

	if resolvedSparkTopology.Spec.AutoScaling.EnableClusterAutoscaler {
		err = checkCmdEnvFolderExists(resolvedSparkTopology.Metadata, CmdEnvClusterAutoscalerHelmChart)
		if err != nil {
			return nil, err
		}
	}

	if resolvedSparkTopology.Spec.AutoScaling.EnableClusterAutoscaler {
		err = awslib.CheckEksCtlCmd("eksctl")
		if err != nil {
			return nil, err
		}
	}

	return resolvedTopology, nil
}

func (t *TopologyHandler) Install(topology framework.Topology) (framework.DeploymentOutput, error) {
	sparkTopology := topology.(*SparkTopology)

	deployment := framework.NewDeployment()

	commandEnvironment := framework.CreateCommandEnvironment(sparkTopology.Metadata.CommandEnvironment)

	if commandEnvironment.Get(CmdEnvSparkOperatorHelmChart) == "" {
		return nil, fmt.Errorf("please provide helm chart file location for Spark Operator")
	}

	if sparkTopology.Spec.AutoScaling.EnableClusterAutoscaler && commandEnvironment.Get(CmdEnvClusterAutoscalerHelmChart) == "" {
		return nil, fmt.Errorf("please provide helm chart file location for Cluster Autoscaler")
	}

	kubelib.CheckHelmOrFatal(commandEnvironment.Get(CmdEnvHelmExecutable))
	if commandEnvironment.GetBoolOrElse(CmdEnvWithMinikube, false) {
		commandEnvironment.Set(CmdEnvKubeConfig, kubelib.GetKubeConfigPath())
		deployment.AddStep("minikubeProfile", "Set Minikube Profile", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
			_, err := resource.MinikubeExec("profile", sparkTopology.Spec.EKS.ClusterName)
			return framework.NewDeploymentStepOutput(), err
		})

		deployment.AddStep("minikubeStart", "Start Minikube Cluster", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
			_, err := resource.MinikubeExec("start", "--memory", "4096") // TODO make memory size configurable
			return framework.NewDeploymentStepOutput(), err
		})

		deployment.AddStep("minikubeStatus", "Check Minikube Status", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
			_, err := resource.MinikubeExec("status")
			return framework.NewDeploymentStepOutput(), err
		})
	} else {
		deployment.AddStep("createS3Bucket", "Create S3 bucket", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
			sparkTopology := t.(*SparkTopology)
			err := awslib.CreateS3Bucket(sparkTopology.Spec.Region, sparkTopology.Spec.S3BucketName)
			return framework.DeploymentStepOutput{"bucketName": sparkTopology.Spec.S3BucketName}, err
		})

		deployment.AddStep("createInstanceIAMRole", "Create EKS instance IAM role", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
			sparkTopology := t.(*SparkTopology)
			roleName := CreateInstanceIAMRole(*sparkTopology)
			return framework.DeploymentStepOutput{"roleName": roleName}, nil
		})

		deployment.AddStep("createEKSCluster", "Create EKS cluster", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
			sparkTopology := t.(*SparkTopology)
			err := resource.CreateEksCluster(sparkTopology.Spec.Region, sparkTopology.Spec.VpcId, sparkTopology.Spec.EKS)
			if err != nil {
				return framework.NewDeploymentStepOutput(), err
			}
			clusterSummary, err := resource.DescribeEksCluster(sparkTopology.Spec.Region, sparkTopology.Spec.EKS.ClusterName)
			if err != nil {
				return framework.NewDeploymentStepOutput(), err
			}
			return framework.DeploymentStepOutput{"oidcIssuer": clusterSummary.OidcIssuer}, nil
		})

		deployment.AddStep("createNodeGroups", "Create node groups", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
			sparkTopology := t.(*SparkTopology)
			stepOutput := c.GetStepOutput("createInstanceIAMRole")
			roleName := stepOutput["roleName"].(string)
			if roleName == "" {
				return framework.NewDeploymentStepOutput(), fmt.Errorf("failed to get role name from previous step")
			}
			roleArn, err := awslib.GetIAMRoleArnByName(sparkTopology.Spec.Region, roleName)
			if err != nil {
				return framework.NewDeploymentStepOutput(), err
			}
			for _, nodeGroup := range sparkTopology.Spec.NodeGroups {
				err := resource.CreateNodeGroup(sparkTopology.Spec.Region, sparkTopology.Spec.EKS.ClusterName, nodeGroup, roleArn)
				if err != nil {
					return framework.NewDeploymentStepOutput(), err
				}
			}
			return framework.NewDeploymentStepOutput(), nil
		})

		if sparkTopology.Spec.AutoScaling.EnableClusterAutoscaler {
			deployment.AddStep("enableIamOidcProvider", "Enable IAM OIDC Provider", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
				sparkTopology := t.(*SparkTopology)
				awslib.RunEksCtlCmd("eksctl",
					[]string{"utils", "associate-iam-oidc-provider",
						"--region", sparkTopology.Spec.Region,
						"--cluster", sparkTopology.Spec.EKS.ClusterName,
						"--approve"})
				return framework.NewDeploymentStepOutput(), nil
			})

			deployment.AddStep("createClusterAutoscalerIAMRole", "Create Cluster Autoscaler IAM role", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
				sparkTopology := t.(*SparkTopology)
				oidcIssuer := c.GetStepOutput("createEKSCluster")["oidcIssuer"].(string)
				idStr := "id/"
				index := strings.LastIndex(strings.ToLower(oidcIssuer), idStr)
				if index == -1 {
					return framework.NewDeploymentStepOutput(), fmt.Errorf("invalid OIDC issuer: %s", oidcIssuer)
				}
				oidcId := oidcIssuer[index + len(idStr):]
				roleName, err := CreateClusterAutoscalerIAMRole(*sparkTopology, oidcId)
				return framework.DeploymentStepOutput{"roleName": roleName}, err
			})

			deployment.AddStep("createClusterAutoscalerIAMServiceAccount", "Create Cluster Autoscaler IAM service account", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
				sparkTopology := t.(*SparkTopology)
				roleName := c.GetStepOutput("createClusterAutoscalerIAMRole")["roleName"].(string)
				err := CreateClusterAutoscalerIAMServiceAccount(commandEnvironment, *sparkTopology, roleName)
				return framework.NewDeploymentStepOutput(), err
			})

			deployment.AddStep("createClusterAutoscalerTagsOnNodeGroup", "Create Cluster Autoscaler tags on node groups", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
				sparkTopology := t.(*SparkTopology)
				for _, nodeGroup := range sparkTopology.Spec.NodeGroups {
					err := awslib.CreateOrUpdateClusterAutoscalerTagsOnNodeGroup(sparkTopology.Spec.Region, sparkTopology.Spec.EKS.ClusterName, nodeGroup.Name)
					if err != nil {
						return framework.NewDeploymentStepOutput(), err
					}
				}
				return framework.NewDeploymentStepOutput(), nil
			})

			deployment.AddStep("deployClusterAutoscaler", "Deploy Cluster Autoscaler", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
				sparkTopology := t.(*SparkTopology)
				DeployClusterAutoscaler(commandEnvironment, *sparkTopology)
				return framework.NewDeploymentStepOutput(), nil
			})
		}
	}

	if commandEnvironment.Get(CmdEnvNginxHelmChart) != "" {
		deployment.AddStep("deployNginxIngressController", "Deploy Nginx ingress controller", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
			sparkTopology := t.(*SparkTopology)
			return DeployNginxIngressController(commandEnvironment, *sparkTopology), nil
		})
	}

	deployment.AddStep("deploySparkOperator", "Deploy Spark Operator", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
		sparkTopology := t.(*SparkTopology)
		DeploySparkOperator(commandEnvironment, *sparkTopology)
		return framework.NewDeploymentStepOutput(), nil
	})

	err := deployment.RunSteps(sparkTopology)
	return deployment.GetOutput(), err
}

func (t *TopologyHandler) Uninstall(topology framework.Topology) (framework.DeploymentOutput, error) {
	sparkTopology := topology.(*SparkTopology)

	commandEnvironment := framework.CreateCommandEnvironment(sparkTopology.Metadata.CommandEnvironment)
	deployment := framework.NewDeployment()

	if commandEnvironment.GetBoolOrElse(CmdEnvWithMinikube, false) {
		deployment.AddStep("minikubeProfile", "Set Minikube Profile", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
			_, err := resource.MinikubeExec("profile", sparkTopology.Spec.EKS.ClusterName)
			return framework.NewDeploymentStepOutput(), err
		})

		deployment.AddStep("minikubeStop", "Stop Minikube Cluster", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
			_, err := resource.MinikubeExec("stop")
			return framework.NewDeploymentStepOutput(), err
		})

		deployment.AddStep("minikubeDelete", "Delete Minikube Cluster", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
			_, err := resource.MinikubeExec("delete")
			return framework.NewDeploymentStepOutput(), err
		})
	} else {
		deployment.AddStep("deleteOidcProvider", "Delete OIDC Provider", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
			sparkTopology := t.(*SparkTopology)
			clusterSummary, err := resource.DescribeEksCluster(sparkTopology.Spec.Region, sparkTopology.Spec.EKS.ClusterName)
			if err != nil {
				log.Printf("[WARN] Cannot delete OIDC provider, failed to get EKS cluster %s in regsion %s: %s", sparkTopology.Spec.EKS.ClusterName, sparkTopology.Spec.Region, err.Error())
				return framework.NewDeploymentStepOutput(), nil
			}
			if clusterSummary.OidcIssuer != "" {
				log.Printf("Deleting OIDC Identity Provider %s", clusterSummary.OidcIssuer)
				err = awslib.DeleteOidcProvider(sparkTopology.Spec.Region, clusterSummary.OidcIssuer)
				if err != nil {
					log.Printf("[WARN] Failed to delete OIDC provider %s: %s", clusterSummary.OidcIssuer, err.Error())
					return framework.NewDeploymentStepOutput(), nil
				}
				log.Printf("Deleted OIDC Identity Provider %s", clusterSummary.OidcIssuer)
			}
			return framework.NewDeploymentStepOutput(), nil
		})
		deployment.AddStep("deleteNodeGroups", "Delete Node Groups", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
			sparkTopology := t.(*SparkTopology)
			for _, nodeGroup := range sparkTopology.Spec.NodeGroups {
				err := awslib.DeleteNodeGroup(sparkTopology.Spec.Region, sparkTopology.Spec.EKS.ClusterName, nodeGroup.Name)
				if err != nil {
					return framework.NewDeploymentStepOutput(), err
				}
			}
			return framework.NewDeploymentStepOutput(), nil
		})

		deployment.AddStep("deleteLoadBalancer", "Delete Load Balancer", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
			sparkTopology := t.(*SparkTopology)
			err := awslib.DeleteLoadBalancerOnEKS(sparkTopology.Spec.Region, sparkTopology.Spec.VpcId, sparkTopology.Spec.EKS.ClusterName, "ingress-nginx")
			return framework.NewDeploymentStepOutput(), err
		})

		deployment.AddStep("deleteEKSCluster", "Delete EKS Cluster", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
			sparkTopology := t.(*SparkTopology)
			DeleteEKSCluster(sparkTopology.Spec.Region, sparkTopology.Spec.EKS.ClusterName)
			return framework.NewDeploymentStepOutput(), nil
		})
	}

	err := deployment.RunSteps(sparkTopology)
	return deployment.GetOutput(), err
}

func (t *TopologyHandler) PrintUsageExample(topology framework.Topology, deploymentOutput framework.DeploymentOutput) {
	sparkTopology := topology.(*SparkTopology)

	if _, ok := deploymentOutput.Output()["minikubeStart"]; ok {
		printExampleCommandToRunSparkOnMinikube(*sparkTopology)
	} else {
		var loadBalancerUrls []string
		loadBalancerUrls = deploymentOutput.Output()["deployNginxIngressController"]["loadBalancerUrls"].([]string)
		if len(loadBalancerUrls) > 0 {
			printExampleCommandToRunSparkOnAws(loadBalancerUrls[0], *sparkTopology)
		} else {
			log.Printf("Did not find load balancer url, cannot print usage example command")
		}
	}
}

func printExampleCommandToRunSparkOnMinikube(topology SparkTopology) {
	userName := topology.Spec.ApiGateway.UserName
	userPassword := topology.Spec.ApiGateway.UserPassword
	url := "localhost"

	str := `
------------------------------
Example using sparkcli to run Java Spark application (IMPORTANT: this contains password, please not print out this if there is security concern):
------------------------------
Step 1: Run "minikube tunnel" in another terminal to set up tunnel to minikube. Wait until "minikube tunnel" started.
Step 2: Run: ` + sparkcliJavaExampleCommandFormat
	log.Printf(str, userName, userPassword, url)
}

func printExampleCommandToRunSparkOnAws(url string, topology SparkTopology) {
	userName := topology.Spec.ApiGateway.UserName
	userPassword := topology.Spec.ApiGateway.UserPassword

	file, err := os.CreateTemp("", "pyspark.*.py")
	if err != nil {
		file.Close()
		log.Printf("Failed to create temp file to write example Spark appliation file: %s", err.Error())
	}
	file.Close()

	exampleSparkFileContent := `
from pyspark import SparkContext, SparkConf

if __name__ == "__main__":
  appName = "Word Count - Python"
  print("Running: " + appName)

  conf = SparkConf().setAppName(appName)
  sc = SparkContext(conf=conf)

  words = sc.parallelize(["apple", "banana", "hello", "apple"])

  wordCounts = words.map(lambda word: (word, 1)).reduceByKey(lambda a,b:a+b)

  def printData(x):
    print(x)

  wordCounts.foreach(printData)
`
	err = ioutil.WriteFile(file.Name(), []byte(exampleSparkFileContent), fs.ModePerm)
	if err != nil {
		log.Printf("Failed to write temp file %s: %s", file.Name(), err.Error())
		return
	}

	str := `
------------------------------
Example using sparkcli to run Java Spark application (IMPORTANT: this contains password, please not print out this if there is security concern):
------------------------------
` + sparkcliJavaExampleCommandFormat
	log.Printf(str, userName, userPassword, url)

	anotherStr := `
------------------------------
Another example using sparkcli to run Python Spark application from local file (IMPORTANT: this contains password, please not print out this if there is security concern):
------------------------------
` + sparkcliPythonExampleCommandFormat
	log.Printf(anotherStr, userName, userPassword, url, file.Name())
}

func createSparkTopologyTemplate() SparkTopology {
	namePrefix := "{{ or .Values.namePrefix `my` }}"
	s3BucketName := "{{ or .Values.s3BucketName .DefaultS3BucketName }}"

	topology := CreateDefaultSparkTopology(namePrefix, s3BucketName)

	topology.Spec.Region = "{{ or .Values.region `us-west-1` }}"
	topology.Spec.VpcId = "{{ or .Values.vpcId .DefaultVpcId }}"
	topology.Spec.ApiGateway.UserPassword = "{{ .Values.apiUserPassword }}"

	topology.Metadata.CommandEnvironment[CmdEnvHelmExecutable] = "{{ or .Env.helmExecutable `helm` }}"
	topology.Metadata.CommandEnvironment[CmdEnvWithMinikube] = "{{ or .Env.withMinikube `false` }}"
	topology.Metadata.CommandEnvironment[CmdEnvNginxHelmChart] = "{{ or .Env.nginxHelmChart `ingress-nginx/charts/ingress-nginx` }}"
	topology.Metadata.CommandEnvironment[CmdEnvSparkOperatorHelmChart] = "{{ or .Env.sparkOperatorHelmChart `spark-operator-service/charts/spark-operator-chart` }}"
	topology.Metadata.CommandEnvironment[CmdEnvClusterAutoscalerHelmChart] = "{{ or .Env.clusterAutoscalerHelmChart `cluster-autoscaler/charts/cluster-autoscaler` }}"
	topology.Metadata.CommandEnvironment[CmdEnvKubeConfig] = "{{ or .Env.kubeConfig `` }}"

	topology.Metadata.Notes["apiUserPassword"] = "Please make sure to provide API gateway user password when deploying the topology, e.g. --set apiUserPassword=your-password"

	return topology
}

func checkCmdEnvFolderExists(metadata framework.TopologyMetadata, cmdEnvKey string) error {
	cmdEnvValue := metadata.CommandEnvironment[cmdEnvKey]
	if cmdEnvValue == "" {
		return fmt.Errorf("Metadata.CommandEnvironment[\"%s\"] is empty", cmdEnvKey)
	}
	if _, err := os.Stat(cmdEnvValue); os.IsNotExist(err) {
		return fmt.Errorf("folder not exists (specified in Metadata.CommandEnvironment[\"%s\"]=\"%s\")", cmdEnvKey, cmdEnvValue)
	}
	return nil
}

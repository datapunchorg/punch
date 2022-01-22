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
	"text/template"
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

	return resolvedTopology, nil
}

func (t *TopologyHandler) Install(topology framework.Topology) (framework.DeploymentOutput, error) {
	sparkTopology := topology.(*SparkTopology)

	deployment := framework.NewDeployment()

	commandEnvironment := framework.CreateCommandEnvironment(sparkTopology.Metadata.CommandEnvironment)

	if commandEnvironment.Get(CmdEnvSparkOperatorHelmChart) == "" {
		return nil, fmt.Errorf("please provide helm chart file location for Spark Operator")
	}

	kubelib.CheckHelmOrFatal(commandEnvironment.Get(CmdEnvHelmExecutable))

	if !commandEnvironment.GetBoolOrElse(CmdEnvIgnoreCreateEKS, false) {
		deployment.AddStep("createS3Bucket", "Create S3 bucket", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
			sparkTopology := t.(*SparkTopology)
			err := awslib.CreateS3Bucket(sparkTopology.Spec.Region, sparkTopology.Spec.S3BucketName)
			return framework.NewDeploymentStepOutput(), err
		})

		deployment.AddStep("createInstanceIAMRole", "Create EKS instance IAM role", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
			sparkTopology := t.(*SparkTopology)
			roleName := CreateInstanceIAMRole(*sparkTopology)
			return framework.DeploymentStepOutput{"roleName": roleName}, nil
		})

		deployment.AddStep("createEKSCluster", "Create EKS cluster", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
			sparkTopology := t.(*SparkTopology)
			err := resource.CreateEksCluster(sparkTopology.Spec.Region, sparkTopology.Spec.VpcId, sparkTopology.Spec.EKS)
			return framework.NewDeploymentStepOutput(), err
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

	deployment := framework.NewDeployment()

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
		err := awslib.DeleteEKSCluster(sparkTopology.Spec.Region, sparkTopology.Spec.EKS.ClusterName)
		return framework.NewDeploymentStepOutput(), err
	})

	err := deployment.RunSteps(sparkTopology)
	return deployment.GetOutput(), err
}

func (t *TopologyHandler) PrintUsageExample(topology framework.Topology, deploymentOutput framework.DeploymentOutput) {
	sparkTopology := topology.(*SparkTopology)

	var loadBalancerUrls []string
	loadBalancerUrls = deploymentOutput.Output()["deployNginxIngressController"]["loadBalancerUrls"].([]string)
	if len(loadBalancerUrls) > 0 {
		printExampleCommandToRunSpark(loadBalancerUrls[0], *sparkTopology)
	}
}

func printExampleCommandToRunSpark(url string, topology SparkTopology) {
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

	str2 := `sparkcli --user %s --password %s --insecure --url https://%s/sparkapi/v1 submit --class org.apache.spark.examples.SparkPi --image ghcr.io/datapunchorg/spark:spark-3.2-1642867779 --spark-version 3.2 --driver-memory 512m --executor-memory 512m local:///opt/spark/examples/jars/spark-examples_2.12-3.2.2-SNAPSHOT.jar`
	log.Printf("\n------------------------------\nExample using sparkcli to run Java Spark application (IMPORTANT: this contains password, please not print out this if there is security concern):\n------------------------------\n" + str2, userName, userPassword, url)

	str3 := `sparkcli --user %s --password %s --insecure --url https://%s/sparkapi/v1 submit --image ghcr.io/datapunchorg/spark:pyspark-3.2-1642867779 --spark-version 3.2 --driver-memory 512m --executor-memory 512m %s`
	log.Printf("\n------------------------------\nAnother example using sparkcli to run Python Spark application from local file (IMPORTANT: this contains password, please not print out this if there is security concern):\n------------------------------\n" + str3, userName, userPassword, url, file.Name())
}

func createSparkTopologyTemplate() SparkTopology {
	namePrefix := "{{ or .Values.namePrefix `my` }}"
	s3BucketName := "{{ or .Values.s3BucketName .DefaultS3BucketName }}"

	topology := CreateDefaultSparkTopology(namePrefix, s3BucketName)

	topology.Spec.Region = "{{ or .Values.region `us-west-1` }}"
	topology.Spec.VpcId = "{{ or .Values.vpcId .DefaultVpcId }}"
	topology.Spec.ApiGateway.UserPassword = "{{ .Values.apiUserPassword }}"

	topology.Metadata.CommandEnvironment[CmdEnvHelmExecutable] = "{{ or .Env.helmExecutable `helm` }}"
	topology.Metadata.CommandEnvironment[CmdEnvIgnoreCreateEKS] = "{{ or .Env.ignoreCreateEKS `false` }}"
	topology.Metadata.CommandEnvironment[CmdEnvNginxHelmChart] = "{{ or .Env.nginxHelmChart `ingress-nginx/charts/ingress-nginx` }}"
	topology.Metadata.CommandEnvironment[CmdEnvSparkOperatorHelmChart] = "{{ or .Env.sparkOperatorHelmChart `spark-operator-service/charts/spark-operator-chart` }}"
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
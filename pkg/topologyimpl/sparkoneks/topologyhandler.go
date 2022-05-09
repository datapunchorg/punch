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

package main

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/datapunchorg/punch/pkg/awslib"
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/topologyimpl/eks"
	"gopkg.in/yaml.v3"
)

const (
	sparkcliJavaExampleCommandFormat   = `./sparkcli --user %s --password %s --insecure --url %s/sparkapi/v1 submit --class org.apache.spark.examples.SparkPi --spark-version 3.2 --driver-memory 512m --executor-memory 512m local:///opt/spark/examples/jars/spark-examples_2.12-3.2.1.jar`
	sparkcliPythonExampleCommandFormat = `./sparkcli --user %s --password %s --insecure --url %s/sparkapi/v1 submit --spark-version 3.2 --driver-memory 512m --executor-memory 512m %s`
)

type TopologyHandler struct {
}

var Handler TopologyHandler

func (t *TopologyHandler) Generate() (framework.Topology, error) {
	topology := GenerateSparkOnEksTopology()
	return &topology, nil
}

func (t *TopologyHandler) Parse(yamlContent []byte) (framework.Topology, error) {
	result := CreateDefaultSparkEksTopology(framework.DefaultNamePrefix, eks.ToBeReplacedS3BucketName)
	err := yaml.Unmarshal(yamlContent, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML (%s): \n%s", err.Error(), string(yamlContent))
	}
	return &result, nil
}

func (t *TopologyHandler) Validate(topology framework.Topology, phase string) (framework.Topology, error) {
	currentTopology := topology.(*SparkOnEksTopology)

	if strings.EqualFold(phase, framework.PhaseBeforeInstall) {
		if currentTopology.Spec.ApiGateway.UserPassword == "" || currentTopology.Spec.ApiGateway.UserPassword == framework.TemplateNoValue {
			return nil, fmt.Errorf("spec.apiGateway.userPassword is emmpty, please provide the value for the password")
		}
	}

	err := checkCmdEnvFolderExists(currentTopology.Metadata, eks.CmdEnvNginxHelmChart)
	if err != nil {
		return nil, err
	}

	err = checkCmdEnvFolderExists(currentTopology.Metadata, CmdEnvSparkOperatorHelmChart)
	if err != nil {
		return nil, err
	}

	if currentTopology.Spec.EksSpec.AutoScaling.EnableClusterAutoscaler {
		err = checkCmdEnvFolderExists(currentTopology.Metadata, eks.CmdEnvClusterAutoscalerHelmChart)
		if err != nil {
			return nil, err
		}
	}

	if currentTopology.Spec.EksSpec.AutoScaling.EnableClusterAutoscaler {
		err = awslib.CheckEksCtlCmd("eksctl")
		if err != nil {
			return nil, err
		}
	}

	return topology, nil
}

func (t *TopologyHandler) Install(topology framework.Topology) (framework.DeploymentOutput, error) {
	currentTopology := topology.(*SparkOnEksTopology)

	commandEnvironment := framework.CreateCommandEnvironment(currentTopology.Metadata.CommandEnvironment)

	deployment, err := BuildInstallDeployment(currentTopology.Spec, commandEnvironment)
	if err != nil {
		return deployment.GetOutput(), err
	}

	err = deployment.Run()
	return deployment.GetOutput(), err
}

func (t *TopologyHandler) Uninstall(topology framework.Topology) (framework.DeploymentOutput, error) {
	currentTopology := topology.(*SparkOnEksTopology)

	commandEnvironment := framework.CreateCommandEnvironment(currentTopology.Metadata.CommandEnvironment)

	deployment, err := BuildUninstallDeployment(currentTopology.Spec, commandEnvironment)
	if err != nil {
		return deployment.GetOutput(), err
	}

	err = deployment.Run()
	return deployment.GetOutput(), err
}

func (t *TopologyHandler) PrintUsageExample(topology framework.Topology, deploymentOutput framework.DeploymentOutput) {
	currentTopology := topology.(*SparkOnEksTopology)

	loadBalancerUrl := deploymentOutput.Output()["deployNginxIngressController"]["loadBalancerPreferredUrl"].(string)
	if loadBalancerUrl != "" {
		if _, ok := deploymentOutput.Output()["minikubeStart"]; ok {
			printExampleCommandToRunSparkOnMinikube(loadBalancerUrl, *currentTopology)
		} else {
			printExampleCommandToRunSparkOnAws(loadBalancerUrl, *currentTopology)
		}
	} else {
		log.Printf("Did not find load balancer url, cannot print usage example command")
	}
}

func printExampleCommandToRunSparkOnMinikube(url string, topology SparkOnEksTopology) {
	userName := topology.Spec.ApiGateway.UserName
	userPassword := topology.Spec.ApiGateway.UserPassword

	str := `
------------------------------
Example using sparkcli to run Java Spark application (IMPORTANT: this contains password, please not print out this if there is security concern):
------------------------------
Run: ` + sparkcliJavaExampleCommandFormat
	log.Printf(str, userName, userPassword, url)
}

func printExampleCommandToRunSparkOnAws(url string, topology SparkOnEksTopology) {
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

func BuildInstallDeployment(topologySpec SparkOnEksTopologySpec, commandEnvironment framework.CommandEnvironment) (framework.Deployment, error) {
	deployment, err := eks.CreateInstallDeployment(topologySpec.EksSpec, commandEnvironment)
	if err != nil {
		return nil, err
	}

	deployment.AddStep("deploySparkOperator", "Deploy Spark Operator", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
		DeploySparkOperator(commandEnvironment, topologySpec)
		return framework.NewDeploymentStepOutput(), nil
	})

	deployment.AddStep("deploySparkHistoryServer", "Deploy Spark History Server", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
		DeployHistoryServer(commandEnvironment, topologySpec)
		return framework.NewDeploymentStepOutput(), nil
	})

	return deployment, nil
}

func BuildUninstallDeployment(topologySpec SparkOnEksTopologySpec, commandEnvironment framework.CommandEnvironment) (framework.Deployment, error) {
	deployment, err := eks.CreateUninstallDeployment(topologySpec.EksSpec, commandEnvironment)
	return deployment, err
}

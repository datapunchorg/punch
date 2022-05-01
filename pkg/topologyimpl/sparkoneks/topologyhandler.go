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

package sparkoneks

import (
	"fmt"
	"github.com/datapunchorg/punch/pkg/awslib"
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/topologyimpl/eks"
	"gopkg.in/yaml.v3"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

const (
	sparkcliJavaExampleCommandFormat   = `./sparkcli --user %s --password %s --insecure --url https://%s/sparkapi/v1 submit --class org.apache.spark.examples.SparkPi --image ghcr.io/datapunchorg/spark:spark-3.2.1-1643336295 --spark-version 3.2 --driver-memory 512m --executor-memory 512m local:///opt/spark/examples/jars/spark-examples_2.12-3.2.1.jar`
	sparkcliPythonExampleCommandFormat = `./sparkcli --user %s --password %s --insecure --url https://%s/sparkapi/v1 submit --image ghcr.io/datapunchorg/spark:pyspark-3.2.1-1643336295 --spark-version 3.2 --driver-memory 512m --executor-memory 512m %s`
)

func init() {
	framework.DefaultTopologyHandlerManager.AddHandler(KindSparkTopology, &TopologyHandler{})
	// TODO delete SparkOnK8s in the future
	framework.DefaultTopologyHandlerManager.AddHandler("SparkOnK8s", &TopologyHandler{})
}

type TopologyHandler struct {
}

func (t *TopologyHandler) Generate() (framework.Topology, error) {
	topology := GenerateSparkTopology()
	return &topology, nil
}

func (t *TopologyHandler) Parse(yamlContent []byte) (framework.Topology, error) {
	result := CreateDefaultSparkTopology(DefaultNamePrefix, eks.ToBeReplacedS3BucketName)
	err := yaml.Unmarshal(yamlContent, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML (%s): \n%s", err.Error(), string(yamlContent))
	}
	return &result, nil
}

func (t *TopologyHandler) Validate(topology framework.Topology, phase string) (framework.Topology, error) {
	resolvedSpecificTopology := topology.(*SparkTopology)

	if strings.EqualFold(phase, framework.PhaseBeforeInstall) {
		if resolvedSpecificTopology.Spec.ApiGateway.UserPassword == "" || resolvedSpecificTopology.Spec.ApiGateway.UserPassword == framework.TemplateNoValue {
			return nil, fmt.Errorf("spec.apiGateway.userPassword is emmpty, please provide the value for the password")
		}
	}

	err := checkCmdEnvFolderExists(resolvedSpecificTopology.Metadata, eks.CmdEnvNginxHelmChart)
	if err != nil {
		return nil, err
	}

	err = checkCmdEnvFolderExists(resolvedSpecificTopology.Metadata, CmdEnvSparkOperatorHelmChart)
	if err != nil {
		return nil, err
	}

	if resolvedSpecificTopology.Spec.EksSpec.AutoScaling.EnableClusterAutoscaler {
		err = checkCmdEnvFolderExists(resolvedSpecificTopology.Metadata, eks.CmdEnvClusterAutoscalerHelmChart)
		if err != nil {
			return nil, err
		}
	}

	if resolvedSpecificTopology.Spec.EksSpec.AutoScaling.EnableClusterAutoscaler {
		err = awslib.CheckEksCtlCmd("eksctl")
		if err != nil {
			return nil, err
		}
	}

	return topology, nil
}

func (t *TopologyHandler) Install(topology framework.Topology) (framework.DeploymentOutput, error) {
	specificTopology := topology.(*SparkTopology)

	commandEnvironment := framework.CreateCommandEnvironment(specificTopology.Metadata.CommandEnvironment)

	deployment, err := BuildInstallDeployment(specificTopology.Spec, commandEnvironment)
	if err != nil {
		return deployment.GetOutput(), err
	}

	err = deployment.Run()
	return deployment.GetOutput(), err
}

func (t *TopologyHandler) Uninstall(topology framework.Topology) (framework.DeploymentOutput, error) {
	specificTopology := topology.(*SparkTopology)

	commandEnvironment := framework.CreateCommandEnvironment(specificTopology.Metadata.CommandEnvironment)

	deployment, err := BuildUninstallDeployment(specificTopology.Spec, commandEnvironment)
	if err != nil {
		return deployment.GetOutput(), err
	}

	err = deployment.Run()
	return deployment.GetOutput(), err
}

func (t *TopologyHandler) PrintUsageExample(topology framework.Topology, deploymentOutput framework.DeploymentOutput) {
	specificTopology := topology.(*SparkTopology)

	var loadBalancerUrls []string
	loadBalancerUrls = deploymentOutput.Output()["deployNginxIngressController"]["loadBalancerUrls"].([]string)
	if len(loadBalancerUrls) > 0 {
		url := loadBalancerUrls[0]
		if _, ok := deploymentOutput.Output()["minikubeStart"]; ok {
			printExampleCommandToRunSparkOnMinikube(url, *specificTopology)
		} else {
			printExampleCommandToRunSparkOnAws(url, *specificTopology)
		}
	} else {
		log.Printf("Did not find load balancer url, cannot print usage example command")
	}
}

func printExampleCommandToRunSparkOnMinikube(url string, topology SparkTopology) {
	userName := topology.Spec.ApiGateway.UserName
	userPassword := topology.Spec.ApiGateway.UserPassword

	str := `
------------------------------
Example using sparkcli to run Java Spark application (IMPORTANT: this contains password, please not print out this if there is security concern):
------------------------------
Run: ` + sparkcliJavaExampleCommandFormat
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

func BuildInstallDeployment(topologySpec SparkTopologySpec, commandEnvironment framework.CommandEnvironment) (framework.Deployment, error) {
	deployment, err := eks.BuildInstallDeployment(topologySpec.EksSpec, commandEnvironment)
	if err != nil {
		return nil, err
	}

	deployment.AddStep("deploySparkOperator", "Deploy Spark Operator", func(c framework.DeploymentContext) (framework.DeploymentStepOutput, error) {
		DeploySparkOperator(commandEnvironment, topologySpec)
		return framework.NewDeploymentStepOutput(), nil
	})

	deployment.AddStep("deploySparkHistoryServer", "Deploy Spark History Server", func(c framework.DeploymentContext) (framework.DeploymentStepOutput, error) {
		DeployHistoryServer(commandEnvironment, topologySpec)
		return framework.NewDeploymentStepOutput(), nil
	})

	return deployment, nil
}

func BuildUninstallDeployment(topologySpec SparkTopologySpec, commandEnvironment framework.CommandEnvironment) (framework.Deployment, error) {
	deployment, err := eks.BuildUninstallDeployment(topologySpec.EksSpec, commandEnvironment)
	return deployment, err
}

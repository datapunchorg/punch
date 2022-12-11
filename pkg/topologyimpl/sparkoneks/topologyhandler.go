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
	"fmt"
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/topologyimpl/eks"
	"gopkg.in/yaml.v3"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
)

const (
	SparkApiRelativeUrl = "/sparkapi/v1"

	sparkcliJavaExampleCommandFormat   = `./sparkcli --user %s --password %s --insecure --url %s submit --class org.apache.spark.examples.SparkPi --spark-version 3.2 --driver-memory 512m --executor-memory 512m local:///opt/spark/examples/jars/spark-examples_2.12-3.2.1.jar`
	sparkcliPythonExampleCommandFormat = `./sparkcli --user %s --password %s --insecure --url %s submit --spark-version 3.2 --driver-memory 512m --executor-memory 512m %s`
)

type TopologyHandler struct {
}

func (t *TopologyHandler) Generate() (framework.Topology, error) {
	topology := GenerateSparkOnEksTopology()
	return &topology, nil
}

func (t *TopologyHandler) Parse(yamlContent []byte) (framework.Topology, error) {
	result := CreateDefaultSparkOnEksTopology(framework.DefaultNamePrefix, eks.ToBeReplacedS3BucketName)
	err := yaml.Unmarshal(yamlContent, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML (%s): \n%s", err.Error(), string(yamlContent))
	}
	return &result, nil
}

func (t *TopologyHandler) Validate(topology framework.Topology, phase string) (framework.Topology, error) {
	currentTopology := topology.(*SparkOnEksTopology)

	err := ValidateSparkComponentSpec(currentTopology.Spec.Spark, currentTopology.Metadata, phase)
	if err != nil {
		return nil, err
	}

	err = eks.ValidateEksTopologySpec(currentTopology.Spec.Eks, currentTopology.Metadata, phase)
	if err != nil {
		return nil, err
	}

	return topology, nil
}

func (t *TopologyHandler) Install(topology framework.Topology) (framework.DeploymentOutput, error) {
	currentTopology := topology.(*SparkOnEksTopology)
	commandEnvironment := framework.CreateCommandEnvironment(currentTopology.Metadata.CommandEnvironment)
	deployment, err := CreateInstallDeployment(currentTopology.Spec, commandEnvironment)
	if err != nil {
		return deployment.GetOutput(), err
	}
	err = deployment.Run()
	return deployment.GetOutput(), err
}

func (t *TopologyHandler) Uninstall(topology framework.Topology) (framework.DeploymentOutput, error) {
	currentTopology := topology.(*SparkOnEksTopology)
	commandEnvironment := framework.CreateCommandEnvironment(currentTopology.Metadata.CommandEnvironment)
	deployment, err := CreateUninstallDeployment(currentTopology.Spec, commandEnvironment)
	if err != nil {
		return deployment.GetOutput(), err
	}
	err = deployment.Run()
	return deployment.GetOutput(), err
}

func (t *TopologyHandler) PrintNotes(topology framework.Topology, deploymentOutput framework.DeploymentOutput) {
	currentTopology := topology.(*SparkOnEksTopology)

	loadBalancerUrl := ""
	outputValue := deploymentOutput.Output()["installNginxIngressController"]["loadBalancerPreferredUrl"]
	if outputValue != nil {
		loadBalancerUrl = outputValue.(string)
	}
	if loadBalancerUrl != "" {
		apiGatewayUrl := loadBalancerUrl + SparkApiRelativeUrl
		if _, ok := deploymentOutput.Output()["minikubeStart"]; ok {
			printExampleCommandToRunSparkOnMinikube(apiGatewayUrl, *currentTopology)
		} else {
			printExampleCommandToRunSparkOnAws(apiGatewayUrl, *currentTopology)
		}
	} else {
		log.Printf("Did not find load balancer url, cannot print notes to show usage example command")
	}
}

func printExampleCommandToRunSparkOnMinikube(url string, topology SparkOnEksTopology) {
	userName := topology.Spec.Spark.Gateway.User
	userPassword := topology.Spec.Spark.Gateway.Password

	str := `
------------------------------
Example using sparkcli to run Java Spark application (IMPORTANT: this contains password, please not print out this if there is security concern):
------------------------------
Run: ` + sparkcliJavaExampleCommandFormat
	log.Printf(str, userName, userPassword, url)
}

func printExampleCommandToRunSparkOnAws(url string, topology SparkOnEksTopology) {
	userName := topology.Spec.Spark.Gateway.User
	userPassword := topology.Spec.Spark.Gateway.Password

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

func CreateInstallDeployment(topologySpec SparkOnEksTopologySpec, commandEnvironment framework.CommandEnvironment) (framework.Deployment, error) {
	deployment, err := eks.CreateInstallDeployment(topologySpec.Eks, commandEnvironment)
	if err != nil {
		return nil, err
	}
	deployment.AddStep("deploySparkOperator", "Deploy Spark Operator", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
		err := DeploySparkOperator(commandEnvironment, topologySpec.Spark, topologySpec.Eks.Region, topologySpec.Eks.EksCluster.ClusterName, topologySpec.Eks.S3BucketName)
		return nil, err
	})
	if topologySpec.Spark.HistoryServer.Enable {
		deployment.AddStep("deploySparkHistoryServer", "Deploy Spark History Server", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
			err := DeployHistoryServer(commandEnvironment, topologySpec.Spark, topologySpec.Eks.Region, topologySpec.Eks.EksCluster.ClusterName)
			return nil, err
		})
	}
	return deployment, nil
}

func CreateUninstallDeployment(topologySpec SparkOnEksTopologySpec, commandEnvironment framework.CommandEnvironment) (framework.Deployment, error) {
	deployment, err := eks.CreateUninstallDeployment(topologySpec.Eks, commandEnvironment)
	return deployment, err
}

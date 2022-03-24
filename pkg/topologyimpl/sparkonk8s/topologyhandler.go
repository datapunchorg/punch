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
	"github.com/datapunchorg/punch/pkg/topologyimpl/eks"
	"gopkg.in/yaml.v3"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
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
	result := CreateDefaultSparkTopology(DefaultNamePrefix, eks.ToBeReplacedS3BucketName)
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

	sparkData := eks.CreateEksTemplateData(data)

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

	err = checkCmdEnvFolderExists(resolvedSparkTopology.Metadata, eks.CmdEnvNginxHelmChart)
	if err != nil {
		return nil, err
	}

	err = checkCmdEnvFolderExists(resolvedSparkTopology.Metadata, CmdEnvSparkOperatorHelmChart)
	if err != nil {
		return nil, err
	}

	if resolvedSparkTopology.Spec.EksSpec.AutoScaling.EnableClusterAutoscaler {
		err = checkCmdEnvFolderExists(resolvedSparkTopology.Metadata, eks.CmdEnvClusterAutoscalerHelmChart)
		if err != nil {
			return nil, err
		}
	}

	if resolvedSparkTopology.Spec.EksSpec.AutoScaling.EnableClusterAutoscaler {
		err = awslib.CheckEksCtlCmd("eksctl")
		if err != nil {
			return nil, err
		}
	}

	return resolvedTopology, nil
}

func (t *TopologyHandler) Install(topology framework.Topology) (framework.DeploymentOutput, error) {
	specificTopology := topology.(*SparkTopology)

	commandEnvironment := framework.CreateCommandEnvironment(specificTopology.Metadata.CommandEnvironment)

	deployment, err := BuildInstallDeployment(specificTopology.Spec, commandEnvironment)
	if err != nil {
		return deployment.GetOutput(), err
	}

	err = deployment.RunSteps(specificTopology.GetSpec())
	return deployment.GetOutput(), err
}

func (t *TopologyHandler) Uninstall(topology framework.Topology) (framework.DeploymentOutput, error) {
	specificTopology := topology.(*SparkTopology)

	commandEnvironment := framework.CreateCommandEnvironment(specificTopology.Metadata.CommandEnvironment)

	deployment, err := BuildUninstallDeployment(specificTopology.Spec, commandEnvironment)
	if err != nil {
		return deployment.GetOutput(), err
	}

	err = deployment.RunSteps(specificTopology)
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

// TODO use eks.createEksTopologyTemplate
func createSparkTopologyTemplate() SparkTopology {
	namePrefix := "{{ or .Values.namePrefix `my` }}"
	s3BucketName := "{{ or .Values.s3BucketName .DefaultS3BucketName }}"

	topology := CreateDefaultSparkTopology(namePrefix, s3BucketName)

	eksTemplate := eks.CreateEksTopologyTemplate()

	topology.Spec.EksSpec = eksTemplate.Spec
	topology.Spec.ApiGateway.UserPassword = "{{ .Values.apiUserPassword }}"

	topology.Metadata.CommandEnvironment = eksTemplate.Metadata.CommandEnvironment
	topology.Metadata.CommandEnvironment[CmdEnvSparkOperatorHelmChart] = "{{ or .Env.sparkOperatorHelmChart `spark-operator-service/charts/spark-operator-chart` }}"

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

func BuildInstallDeployment(topologySpec SparkTopologySpec, commandEnvironment framework.CommandEnvironment) (framework.DeploymentImpl, error) {
	deployment, err := eks.BuildInstallDeployment(topologySpec.EksSpec, commandEnvironment)
	if err != nil {
		return deployment, err
	}

	deployment.AddStep("deploySparkOperator", "Deploy Spark Operator", func(c framework.DeploymentContext) (framework.DeploymentStepOutput, error) {
		DeploySparkOperator(commandEnvironment, topologySpec)
		return framework.NewDeploymentStepOutput(), nil
	})

	return deployment, nil
}

func BuildUninstallDeployment(topologySpec SparkTopologySpec, commandEnvironment framework.CommandEnvironment) (framework.DeploymentImpl, error) {
	deployment, err := eks.BuildUninstallDeployment(topologySpec.EksSpec, commandEnvironment)
	return deployment, err
}

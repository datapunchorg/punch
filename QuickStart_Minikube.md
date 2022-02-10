
## Pre-requisite (One Time Setting)

### Install Homebrew

https://docs.brew.sh/Installation

```
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install.sh)"
```

### Install Helm

https://helm.sh/docs/intro/install/

```
brew install helm
```

### Install Docker Desktop and Increase Its Memory

Step 1. Follow this instruction to install Docker Desktop: https://docs.docker.com/desktop/mac/install/

Step 2: Increase memory to 5G in Docker Desktop: [instructions](docs/IncreaseDockerMemory.md)

### Install Minikube

Only do step 1 "Installation" in https://minikube.sigs.k8s.io/docs/start

```
brew install minikube
```

## Run Spark Application

### Install SparkOnK8s on Minikube

Run punch command:

```
./punch install SparkOnK8s --env withMinikube=true --set apiUserPassword=password1 --print-usage-example
```

The upper `punch install` will print out example commands to run Spark application when it finishes. 
Please check the console output. Also see following section for how to run Spark application.

### Run Spark Application

Establish a tunnel in a separate shell window:

```
minikube tunnel
```

`minikube tunnel` may ask your computer account password since it needs privilege to expose network ports. Please 
wait until `minikube tunnel` starts successfully.

Then set the load balancer domain name as below:

```
export LB_NAME=localhost
```

Run sparkcli:

```
./sparkcli --user user1 --password password1 --insecure --url https://$LB_NAME/sparkapi/v1 submit --class org.apache.spark.examples.SparkPi --image ghcr.io/datapunchorg/spark:spark-3.2.1-1643336295 --spark-version 3.2 --driver-memory 512m --executor-memory 512m local:///opt/spark/examples/jars/spark-examples_2.12-3.2.1.jar
```


### Uninstall SparkOnK8s on Minikube

Run punch command:

```
./punch uninstall SparkOnK8s --env withMinikube=true --set apiUserPassword=password1
```

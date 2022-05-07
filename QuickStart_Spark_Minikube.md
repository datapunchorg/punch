Note: this document only for running `punch` on Mac.

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

### Get punch Distribution

Run `make release` in the root of this project, it will generate `dist` folder which contains `punch` and `sparkcli` binaries.

Or download a pre-built zip file and unzip it:
```
curl -L -o dist.zip https://github.com/datapunchorg/punch/releases/download/0.3.0/dist.zip
unzip dist.zip
```

In the terminal, go the `dist` folder.

## Deploy Spark on K8s and Run Spark Application

### Install SparkOnEks on Minikube

Run punch command:

```
./punch install SparkOnEks --env withMinikube=true --patch spec.apiGateway.userPassword=password1 --print-usage-example
```

The upper `punch install` will print out example commands to run Spark application when it finishes.
Please check the console output. Also see following section for how to run Spark application.

### Run Spark Application

Then Run `sparkcli`:

```
./sparkcli --user user1 --password password1 --insecure --url https://localhost:32443/sparkapi/v1 submit --class org.apache.spark.examples.SparkPi --spark-version 3.2 --driver-memory 512m --executor-memory 512m local:///opt/spark/examples/jars/spark-examples_2.12-3.2.1.jar
```

### Uninstall SparkOnEks on Minikube

Run punch command:

```
./punch uninstall SparkOnEks --env withMinikube=true
```


## Introduction

This project is to provide one-click experience for people to create Cloud and Kubernetes environment to run their Data Analytics workload
like Apache Spark.

For example, "punch install SparkOnK8s" will create an AWS EKS cluster and a Spark REST Service. Then you could use curl or command line tool
to submit Spark application.

## How to build (on MacBook)

The following command will create `dist` folder and `dist.zip` file for Punch.

```
make release
```

Go to `dist` folder, then check [User Guide](UserGuide.md) to see how to run punch command.

## Quick Start

You could build this project (`make release`) and use `punch` to deploy Spark on [Minikube](https://minikube.sigs.k8s.io/docs/start/), and run Spark application for a quick try.

See [Quick Start Guide](QuickStart_Minikube.md) for details.

## Install using Homebrew

```
brew tap datapunchorg/punch
brew install punch

brew tap datapunchorg/sparkcli
brew install sparkcli
```

## TODO

1. Delete IAM Identity provider when enabled AutoScale
2. Attach tag (e.g. punch-topology=xxx) to AWS resources created by punch
3. Mask password value in helm output (e.g. --set apiGateway.userPassword=xxx)
4. Remove unnecessary argument like "--set apiUserPassword=password1" in "punch uninstall" command
5. Allow patch topology like --patch foo.field1=value1
6. Allow set values by file like --values values.yaml
7. Return HTTP 404 when sparkcli getting a non-existing application
8. Get application error message from Spark Operator
9. Set up convenient tool to benchmark Spark TPC-DS
10. Create public demo (tech news, mailing list)
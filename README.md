
## Introduction

This project provides a fully automated one-click experience for people to create Cloud and Kubernetes environment 
to run their Data Analytics workload like Apache Spark.

For example, command like `punch install SparkOnK8s` will automatically create an AWS EKS cluster, Ingress Controller, 
IAM Roles, Cluster AutoScaler, Spark Operator and a Spark REST Service. Then people could use curl or a command line 
tool (`spark-cli`) to run Spark application without any other manual deployment work.

## How to build (on MacBook)

The following command will create `dist` folder and `dist.zip` file for Punch.

```
make release
```

Go to `dist` folder, then check [User Guide](UserGuide.md) to see how to run punch command.

## Quick Start - Run Spark on Minikube

You could build this project (`make release`) and use `punch` to deploy Spark on [Minikube](https://minikube.sigs.k8s.io/docs/start/), and run Spark application for a quick try.

See [Quick Start Guide](QuickStart_Minikube.md) for details.

## Quick Start - Create EKS Cluster

You could build this project (`make release`) and use `punch` to crate an AWS EKS cluster and play with it.

See [Quick Start Guide](QuickStart_CreateEks.md) for details.


## Install using Homebrew

```
brew tap datapunchorg/punch
brew install punch

brew tap datapunchorg/sparkcli
brew install sparkcli
```

## TODO

1. Rename Resolve to Validate. Test --patch.
2. Attach tag (e.g. punch-topology=xxx) to AWS resources created by punch
3. Mask password value in helm output (e.g. --set apiGateway.userPassword=xxx)
4. Remove unnecessary argument like "--set apiUserPassword=password1" in "punch uninstall" command
5. Allow patch topology like --patch foo.field1=value1
6. Allow set values by file like --values values.yaml
7. Return HTTP 404 when sparkcli getting a non-existing application
8. Get application error message from Spark Operator
9. Set up convenient tool to benchmark Spark TPC-DS
10. Create public demo (tech news, mailing list)

## Supported By

Thanks for support from [JetBrains](https://jb.gg/OpenSourceSupport) with the great development tool and licenses.

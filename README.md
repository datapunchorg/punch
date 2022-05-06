
## Introduction

This project provides a fully automated one-click experience for people to create Cloud and Kubernetes environment 
to run their Data Analytics workload like Apache Spark.

For example, command like `punch install SparkOnEks` will automatically create an AWS EKS cluster, Ingress Controller, 
IAM Roles, Cluster AutoScaler, Spark Operator and a Spark REST Service. Then people could use curl or a command line 
tool (`sparkcli`) to run Spark application without any other manual deployment work.

## How to build (on MacBook)

The following command will create `dist` folder and `dist.zip` file for Punch.

```
make release
```

Go to `dist` folder, then check [User Guide](UserGuide.md) to see how to run `punch` command.

## Quick Start - Run Spark on Minikube

You could build this project (`make release`) and use `punch` to deploy Spark on [Minikube](https://minikube.sigs.k8s.io/docs/start/), and run Spark application for a quick try.

For example, use one command like `punch install SparkOnEks --env withMinikube=true` to deploy a runnable Spark environment on Minikube.

See [Quick Start Guide](QuickStart_Minikube.md) for details.

## User Guide - Run Spark on AWS EKS

Again, use one command like `punch install SparkOnEks` to deploy a runnable Spark environment on EKS.

See [User Guide](UserGuide.md) for more details in section: `Run punch on AWS`.

## Quick Start - Create EKS Cluster

You could build this project (`make release`) and use `punch` to crate an AWS EKS cluster and play with it.

See [Quick Start Guide](QuickStart_CreateEks.md) for details.

## TODO

1. Build Spark image with kafka support
2. Attach tag (e.g. punch-topology=xxx) to AWS resources created by punch
3. Mask password value in helm output (e.g. --set apiGateway.userPassword=xxx)
4. Allow set values by file like --values values.yaml
5. Return HTTP 404 when sparkcli getting a non-existing application
6. Get application error message from Spark Operator
7. Set up convenient tool to benchmark Spark TPC-DS
8. Create public demo (tech news, mailing list)

## Supported By

Thanks for support from [JetBrains](https://jb.gg/OpenSourceSupport) with the great development tool and licenses.

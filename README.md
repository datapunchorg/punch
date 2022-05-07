
## Introduction

This project provides a fully automated one-click tool to create Data Analytics platform in Cloud and Kubernetes environment:

1. Single script to deploy a full stack data platform: Kafka, Hive Metastore, Spark, and Data Ingestion Job.

2. Spark API Gateway to run Spark platform as a service.

3. Extensible design to support customization and new service deployment.

For example, command like `punch install SparkOnEks` will automatically create an AWS EKS cluster, Ingress Controller, 
IAM Roles, Cluster AutoScaler, Spark Operator and a Spark REST Service. Then people could use `curl` or the command line 
tool (`sparkcli`) to run Spark application without any other manual deployment work.

## How to build (on MacBook)

The following command will create `dist` folder and `dist.zip` file for Punch.

```
make release
```

Go to `dist` folder, then check [User Guide](UserGuide.md) to see how to run `punch` command.

## Quick Start - Run Spark with Minikube

You could build this project (`make release`) and use `punch` to deploy Spark on [Minikube](https://minikube.sigs.k8s.io/docs/start/), and run Spark application for a quick try.

For example, use one command like `punch install SparkOnEks --env withMinikube=true` to deploy a runnable Spark environment on Minikube.

See [Quick Start Guide - Run Spark with Minikube](QuickStart_Spark_Minikube.md) for details.

## User Guide - Run Spark on AWS EKS

Again, use one command like `punch install SparkOnEks` to deploy a runnable Spark environment on EKS.

See [User Guide](UserGuide.md) for more details in section: `Run punch on AWS`.

## Quick Start - Create EKS Cluster

You could build this project (`make release`) and use `punch` to crate an AWS EKS cluster and play with it.

See [Quick Start Guide - Create EKS](QuickStart_CreateEks.md) for details.

## Supported By

Thanks for support from [JetBrains](https://jb.gg/OpenSourceSupport) with the great development tool and licenses.

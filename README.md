## Note

This project is not open sourced yet. We are still working on the progess to open source the project. Please not share the code publicly.

## Introduction

This project is to provide one-click experience for people to create Cloud and Kubernetes environment to run their Data Analytics workload
like Apache Spark.

For example, "punch install SparkOnK8s" will create an AWS EKS cluster and a Spark REST Service. Then you could use curl or command line tool
to submit Spark application.

## How to build (on MacBook)

The following command will create a zip file for Punch.

```
make
```

Then check [User Guide](UserGuide.md) to see how to run punch command.

## TODO

1. Create Spark 3.1 image for Iceberg (Spark 3.2 not compatible with Iceberg)
2. Attach tag (e.g. punch-topology=xxx) to AWS resources created by punch
3. Mask password value in helm output (e.g. --set apiGateway.userPassword=xxx)
4. Allow patch topology like --patch foo.field1=value1
5. Allow set values by file like --values values.yaml
6. Support "sparkcli delete xxx"
7. Return HTTP 404 when sparkcli getting a non-existing application

## Note

This project is not open sourced yet. We are still working on the progess to open source the project. Please not share the code publicly.

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

## TODO

1. Attach tag (e.g. punch-topology=xxx) to AWS resources created by punch
2. Mask password value in helm output (e.g. --set apiGateway.userPassword=xxx)
3. Remove unnecessary argument like "--set apiUserPassword=password1" in "punch uninstall" command
4. Allow patch topology like --patch foo.field1=value1
5. Allow set values by file like --values values.yaml
6. Return HTTP 404 when sparkcli getting a non-existing application
7. Get application error message from Spark operator

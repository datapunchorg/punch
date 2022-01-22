
## Pre-requisite to Run punch command

### Set up AWS environment

1. Create AWS account, then download AWS Command Line Interface (https://aws.amazon.com/cli/).

2. Config AWS credential (https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-quickstart.html):

```
aws configure
```

### Install Helm

1. See https://helm.sh/docs/intro/install/

## Run punch command

### How to install SparkOnK8s on AWS

1. Unzip the zip file from Punch, and enter that folder in your terminal.

2. Run punch command:

```
./punch install SparkOnK8s --set apiUserPassword=password1 --print-usage-example
```

The upper punch command will create an EKS cluster and Spark REST Service, then people could submit Spark application via that REST service. Please note:

1. AWS sometime is slow in creating EKS (e.g. 10 or more minutes), please be patient waiting for the punch command to finish.
2. The punch command will print out example commands to submit Spark application in the end. Please pay attention to those messages in console output.

### How to uninstall SparkOnK8s on AWS

1. Run punch command:

```
./punch uninstall SparkOnK8s --set apiUserPassword=password1
```

## How to run Spark application after installing SparkOnK8s

"punch install" in previous section will print out example commands to run Spark application. 
Please check the output from "punch install" command.

## What is sparkcli command, and how to use it

sparkcli is a command line tool to submit Spark application and check status/log. 
It is packaged into the punch zip file, and you could find sparkcli program there.

Following are some examples to run sparkcli:

```
./sparkcli --user user1 --password your_password --insecure --url https://xxx.us-west-1.elb.amazonaws.com/sparkapi/v1 submit --class org.apache.spark.examples.SparkPi --image gcr.io/spark-operator/spark:v3.1.1 --spark-version 3.1.1 --driver-memory 512m --executor-memory 512m local:///opt/spark/examples/jars/spark-examples_2.12-3.1.1.jar

./sparkcli --user user1 --password your_password --insecure --url https://xxx.us-west-1.elb.amazonaws.com/sparkapi/v1 status your_submission_id

./sparkcli --user user1 --password your_password --insecure --url https://xxx.us-west-1.elb.amazonaws.com/sparkapi/v1 log your_submission_id
```

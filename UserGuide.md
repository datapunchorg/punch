
## Pre-requisite to Run punch command

### Install Helm

1. See https://helm.sh/docs/intro/install/

## Run punch on Minikube

### Extra Pre-requisite: Install Docker and Minikube

1. Install Docker Desktop: https://docs.docker.com/desktop/mac/install/
2. Increase memory to 5G in Docker Desktop: [instructions](docs/IncreaseDockerMemory.md)
3. Install Minikube: only do step 1 "Installation" in https://minikube.sigs.k8s.io/docs/start

### How to Install SparkOnK8s on Minikube

1. Run punch command:

```
./punch install SparkOnK8s --env withMinikube=true --set apiUserPassword=password1 --print-usage-example
```

### How to uninstall SparkOnK8s on Minikube

1. Run punch command:

```
./punch uninstall SparkOnK8s --env withMinikube=true
```

## Run punch on AWS

### Extra Pre-requisite: Set up AWS environment

1. Create AWS account, then download AWS Command Line Interface (https://aws.amazon.com/cli/).

2. Config AWS credential (https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-quickstart.html):

```
aws configure
```

### How to install SparkOnK8s on AWS

1. Unzip the zip file from Punch, and enter that folder in your terminal.

2. Run punch command:

```
./punch install SparkOnK8s --set apiUserPassword=password1 --print-usage-example
```

The upper punch command will create an EKS cluster and Spark REST Service, then people could submit Spark application via that REST service. Please note:

1. AWS sometime is slow in creating EKS (e.g. 10 or more minutes), please be patient waiting for the punch command to finish.
2. The punch command will print out example commands to submit Spark application in the end. Please pay attention to those messages in console output.

You could also generate a topology template file, manually modify that file, and then install from it:

```
./punch generate SparkOnK8s -o /tmp/SparkOnK8s.yaml
./punch install -f /tmp/SparkOnK8s.yaml --set apiUserPassword=password1 --print-usage-example
```

### How to uninstall SparkOnK8s on AWS

1. Run punch command:

```
./punch uninstall SparkOnK8s
```

"punch uninstall" will delete the EKS cluster and related load balancer. It will not delete the IMA role/policy,
since the IAM role/policy may be still used by other AWS resources. You could manually delete them from AWS web UI.
In the future, we may add option in "punch uninstall" command to delete those IAM role/policy in a safe way.

## How to run Spark application after installing SparkOnK8s

"punch install" in previous section will print out example commands to run Spark application.
Please check the output from "punch install" command. Also see following section for how to use `sparkcli` command
line tool.

## What is sparkcli command, and how to use it

sparkcli is a command line tool to submit Spark application and check status/log.
It is packaged into the punch `dist.zip` file if you build punch by `make release`. You could also use
[Homebrew](https://brew.sh) to install it on Mac:

```
brew tap datapunchorg/sparkcli
brew install sparkcli
```

To use `sparkcli`, you need to have a Spark API Gateway which is installed by `punch`. 

If SparkOnK8s is installed on minikube, set load balancer domain name as below:

```
export LB_NAME=localhost:32443
```

If SparkOnK8s is installed on AWS, set it as below (replace the value with real load balancer url from `punch install` command output):
```
export LB_NAME=xxx.us-west-1.elb.amazonaws.com
```

After upper steps, now you could follow below examples to run `sparkcli`:

```
./sparkcli --user user1 --password password1 --insecure --url https://$LB_NAME/sparkapi/v1 submit --class org.apache.spark.examples.SparkPi --image ghcr.io/datapunchorg/spark:spark-3.2.1-1643336295 --spark-version 3.2 --driver-memory 512m --executor-memory 512m local:///opt/spark/examples/jars/spark-examples_2.12-3.2.1.jar

./sparkcli --user user1 --password password1 --insecure --url https://$LB_NAME/sparkapi/v1 status your_submission_id

./sparkcli --user user1 --password password1 --insecure --url https://$LB_NAME/sparkapi/v1 log your_submission_id
```

## Advanced Usage

### How to run Spark with Apache Hive

You need to set up your own Hive metastore server, and use it for your Spark application.
[Here](https://techjogging.com/standalone-hive-metastore-presto-docker.html) is an example
to set up Hive for Presto. It will be similar for Spark.

If you do not want to set up Hive metastore server, we recommend using [Apache Iceberg](https://iceberg.apache.org)
to store your metadata and use it in your Spark application.

### How to run Spark with Apache Iceberg

There are many ways to set up Apache Iceberg. Following are steps to use a JDBC database together with Iceberg.

1. Create a database. You could use AWS Web UI to create an RDS database, or just punch command, like following:
```
./punch install Database --set masterUserPassword=password1
```
The upper punch command will create a serverless RDS database, and print out the endpoint URL. Please write down that URL,
which will be used later. The serverless RDS database will be paused by AWS automatically if not used for certain duration.
You might need to modify the capacity for the RDS database in AWS web UI if it is suck in paused state.

2. Run Spark application with Spark config like following example (please replace xxx with your own values if you
copy/paste to run your own application):
```
./sparkcli --user user1 --password password1 --insecure \
--url https://xxx.us-west-1.elb.amazonaws.com/sparkapi/v1 \
submit --image ghcr.io/datapunchorg/spark:pyspark-3.1-1643212945 --spark-version 3.1 \
--driver-memory 512m --executor-memory 512m \
--conf spark.jars=s3a://datapunch-public-01/jars/iceberg-spark3-runtime-0.12.1.jar,s3a://datapunch-public-01/jars/awssdk-url-connection-client-2.17.105.jar,s3a://datapunch-public-01/jars/awssdk-bundle-2.17.105.jar,s3a://datapunch-public-01/jars/mariadb-java-client-2.7.4.jar \
--conf spark.sql.warehouse.dir=s3a://xxx/warehouse \
--conf spark.sql.extensions=org.apache.iceberg.spark.extensions.IcebergSparkSessionExtensions \
--conf spark.sql.catalog.my_catalog=org.apache.iceberg.spark.SparkCatalog \
--conf spark.sql.catalog.my_catalog.type=hadoop \
--conf spark.sql.catalog.my_catalog.warehouse=s3a://xxx/iceberg-warehouse \
--conf spark.sql.catalog.my_catalog.io-impl=org.apache.iceberg.aws.s3.S3FileIO \
--conf spark.sql.catalog.my_catalog.catalog-impl=org.apache.iceberg.jdbc.JdbcCatalog \
--conf spark.sql.catalog.my_catalog.uri=jdbc:mysql://xxx.us-west-1.rds.amazonaws.com:3306/mydb \
--conf spark.sql.catalog.my_catalog.jdbc.verifyServerCertificate=false \
--conf spark.sql.catalog.my_catalog.jdbc.useSSL=true \
--conf spark.sql.catalog.my_catalog.jdbc.user=user1 \
--conf spark.sql.catalog.my_catalog.jdbc.password=password1 \
pyspark-iceberg-example.py
```


#!/bin/bash

# echo commands to the terminal output
set -ex

namePrefix=my
instanceType=t3.xlarge
instanceCount=4

databaseUser=user1
databasePassword=password1
sparkApiGatewayUser=user1
sparkApiGatewayPassword=password1

./punch install RdsDatabase --set namePrefix=$namePrefix \
  --patch spec.masterUserName=$databaseUser --patch spec.masterUserPassword=$databasePassword \
  -o RdsDatabase.output.json

databaseEndpoint=$(jq -r '.output[] | select(.step=="createDatabase").output.endpoint' RdsDatabase.output.json)

./punch install Eks --set namePrefix=$namePrefix \
  --patch Spec.NodeGroups[0].InstanceTypes[0]=$instanceType \
  --patch Spec.NodeGroups[0].MinSize=$instanceCount \
  --patch Spec.NodeGroups[0].MaxSize=$instanceCount \
  --patch Spec.NodeGroups[0].DesiredSize=$instanceCount \
  -o Eks.output.json

./punch install HiveMetastore --set namePrefix=$namePrefix \
  --patch spec.database.externalDb=true \
  --patch spec.database.connectionString=jdbc:postgresql://${databaseEndpoint}:5432/postgres \
  --patch spec.database.user=$databaseUser --patch spec.database.password=$databasePassword \
  -o HiveMetastore.output.json

metastoreUri=$(jq -r '.output[] | select(.step=="installHiveMetastoreServer").output.metastoreLoadBalancerUrls[0]' HiveMetastore.output.json)
metastoreWarehouseDir=$(jq -r '.output[] | select(.step=="installHiveMetastoreServer").output.metastoreWarehouseDir' HiveMetastore.output.json)

./punch install SparkOnEks --set namePrefix=$namePrefix \
  --patch spec.apiGateway.userName=$sparkApiGatewayUser \
  --patch spec.apiGateway.userPassword=$sparkApiGatewayPassword \
  --patch spec.apiGateway.hiveMetastoreUris=$metastoreUri \
  --patch spec.apiGateway.sparkSqlWarehouseDir=$metastoreWarehouseDir \
  --print-usage-example \
  -o SparkOnEks.output.json

apiGatewayLoadBalancerUrl=$(jq -r '.output[] | select(.step=="deployNginxIngressController").output.loadBalancerPreferredUrl' SparkOnEks.output.json)

echo ******************** Running dummy Spark application to test the system ********************

./sparkcli --user $sparkApiGatewayUser --password $sparkApiGatewayPassword --insecure \
  --url ${apiGatewayLoadBalancerUrl}/sparkapi/v1 submit --class org.apache.spark.examples.SparkPi \
  --spark-version 3.2 \
  --driver-memory 512m --executor-memory 512m \
  local:///opt/spark/examples/jars/spark-examples_2.12-3.2.1.jar

echo ******************** Running TPC-DS Spark application ********************

./sparkcli --user $sparkApiGatewayUser --password $sparkApiGatewayPassword --insecure \
  --url ${apiGatewayLoadBalancerUrl}/sparkapi/v1 submit --class org.apache.spark.sql.execution.benchmark.TPCDSQueryBenchmark \
  --spark-version 3.1 \
  --driver-memory 4g --executor-memory 4g --num-executors $instanceCount \
  --conf spark.jars=s3a://datapunch-public-01/jars/spark-core_2.12-3.1-tests.jar,s3a://datapunch-public-01/jars/spark-catalyst_2.12-3.1-tests.jar \
  s3a://datapunch-public-01/jars/spark-sql_2.12-3.1-tests.jar \
  --data-location s3a://my-133628591400-us-west-1/punch/my/warehouse/tpcds_data_1g

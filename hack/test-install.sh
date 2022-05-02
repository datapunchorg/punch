#!/bin/bash

export databaseUser=user1
export databasePassword=password1
export sparkApiGatewayUser=user1
export sparkApiGatewayPassword=password1

# echo commands to the terminal output
set -ex

./punch install RdsDatabase --patch spec.masterUserName=$databaseUser --patch spec.masterUserPassword=$databasePassword \
  -o RdsDatabase.output.json

export databaseEndpoint=$(jq -r '.output[] | select(.step=="createDatabase").output.endpoint' RdsDatabase.output.json)

./punch install HiveMetastore --patch spec.database.externalDb=true \
  --patch spec.database.connectionString=jdbc:postgresql://${databaseEndpoint}:5432/postgres \
  --patch spec.database.user=$databaseUser --patch spec.database.password=$databasePassword \
  -o HiveMetastore.output.json

export metastoreUri=$(jq -r '.output[] | select(.step=="installHiveMetastoreServer").output.metastoreLoadBalancerUrls[0]' HiveMetastore.output.json)
export metastoreWarehouseDir=$(jq -r '.output[] | select(.step=="installHiveMetastoreServer").output.metastoreWarehouseDir' HiveMetastore.output.json)

./punch install SparkOnEks --patch spec.apiGateway.userName=$sparkApiGatewayUser \
  --patch spec.apiGateway.userPassword=$sparkApiGatewayPassword \
  --patch spec.apiGateway.hiveMetastoreUris=$metastoreUri \
  --patch spec.apiGateway.sparkSqlWarehouseDir=$metastoreWarehouseDir \
  --print-usage-example \
  -o SparkOnEks.output.json

export apiGatewayLoadBalancerUrl=$(jq -r '.output[] | select(.step=="deployNginxIngressController").output.loadBalancerPreferredUrl' SparkOnEks.output.json)

./sparkcli --user $sparkApiGatewayUser --password $sparkApiGatewayPassword --insecure \
  --url ${apiGatewayLoadBalancerUrl}/sparkapi/v1 submit --class org.apache.spark.examples.SparkPi \
  --image ghcr.io/datapunchorg/spark:spark-3.2.1-1643336295 --spark-version 3.2 \
  --driver-memory 512m --executor-memory 512m \
  local:///opt/spark/examples/jars/spark-examples_2.12-3.2.1.jar

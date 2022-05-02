#!/bin/bash

export databaseUser=user1
export databasePassword=password1

# echo commands to the terminal output
set -ex

./punch install RdsDatabase --patch spec.masterUserName=$databaseUser --patch spec.masterUserPassword=$databasePassword -o RdsDatabase.output.json

export databaseEndpoint=$(jq -r '.output[] | select(.step=="createDatabase").output.endpoint' RdsDatabase.output.json)

./punch install HiveMetastore --patch spec.database.externalDb=true --patch spec.database.connectionString=jdbc:postgresql://${databaseEndpoint}:5432/postgres --patch spec.database.user=$databaseUser --patch spec.database.password=$databasePassword


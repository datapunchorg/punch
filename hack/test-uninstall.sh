#!/bin/bash

# echo commands to the terminal output
set -x

# Disable exit on non zero exit
set +e

./punch uninstall KafkaWithBridge

./punch uninstall SparkOnEks

./punch uninstall HiveMetastore

./punch uninstall RdsDatabase

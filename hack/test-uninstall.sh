#!/bin/bash

# echo commands to the terminal output
set -x

# Disable exit on non zero exit
set +e

./punch uninstall KafkaBridge

./punch uninstall SparkOnEks

./punch uninstall HiveMetastore

./punch uninstall Eks

./punch uninstall RdsDatabase

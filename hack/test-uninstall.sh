#!/bin/bash

# echo commands to the terminal output
set -x

# Disable exit on non zero exit
set +e

namePrefix=my

./punch uninstall KafkaBridge --set namePrefix=$namePrefix

./punch uninstall SparkOnEks --set namePrefix=$namePrefix

./punch uninstall HiveMetastore --set namePrefix=$namePrefix

./punch uninstall Eks --set namePrefix=$namePrefix

./punch uninstall RdsDatabase --set namePrefix=$namePrefix

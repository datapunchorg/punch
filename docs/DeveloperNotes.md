## TODO

1. Kyuubi: monitor failed or stopped Spark application and restart session
2. Exit 1 in sparkcli on error
3. Build Spark image with kafka support
4. Attach tag (e.g. punch-topology=xxx) to AWS resources created by punch
5. Mask password value in helm output (e.g. --set spark.gateway.password=xxx)
6. Allow set values by file like --values values.yaml
7. Return HTTP 404 when sparkcli getting a non-existing application 
8. Create public demo (tech news, mailing list)
9. Handle EKS node group DEGRADED status like
```
Could not launch On-Demand Instances. InsufficientInstanceCapacity - We currently do not have sufficient t3.xlarge capacity in the Availability Zone you requested (us-west-1c). Our system will be working on provisioning additional capacity. You can currently get t3.xlarge capacity by not specifying an Availability Zone in your request or choosing us-west-1a. Launching EC2 instance failed.
```

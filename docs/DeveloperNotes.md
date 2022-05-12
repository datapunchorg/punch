## TODO

1. Kyuubi: monitor failed or stopped Spark application and restart session
2. Use spark.ui.proxyBase for Spark UI, e.g. spark.ui.proxyBase: /foo, see:
https://stackoverflow.com/questions/56368948/change-root-path-for-spark-web-ui
3. Exit 1 in sparkcli on error
4. Build Spark image with kafka support
5. Attach tag (e.g. punch-topology=xxx) to AWS resources created by punch
6. Mask password value in helm output (e.g. --set spark.gateway.password=xxx)
7. Allow set values by file like --values values.yaml
8. Return HTTP 404 when sparkcli getting a non-existing application
9. Get application error message from Spark Operator
10. Set up convenient tool to benchmark Spark TPC-DS
11. Create public demo (tech news, mailing list)

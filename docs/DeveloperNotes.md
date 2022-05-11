## TODO

1. Use spark.ui.proxyBase for Spark UI, e.g. spark.ui.proxyBase: /foo, see:
https://stackoverflow.com/questions/56368948/change-root-path-for-spark-web-ui
2. Exit 1 in sparkcli on error
3. Build Spark image with kafka support
4. Attach tag (e.g. punch-topology=xxx) to AWS resources created by punch
5. Mask password value in helm output (e.g. --set spark.gateway.password=xxx)
6. Allow set values by file like --values values.yaml
7. Return HTTP 404 when sparkcli getting a non-existing application
8. Get application error message from Spark Operator
9. Set up convenient tool to benchmark Spark TPC-DS
10. Create public demo (tech news, mailing list)

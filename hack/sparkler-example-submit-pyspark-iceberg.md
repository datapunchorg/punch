
### Use Iceberg with AWS Glue and DynamoDB

See https://github.com/apache/iceberg/blob/master/site/docs/aws.md

Spark config example:

```
--conf spark.jars=s3a://foo/iceberg-spark3-runtime-0.12.1.jar,s3a://foo/awssdk-url-connection-client-2.17.105.jar,s3a://foo/awssdk-bundle-2.17.105.jar \
--conf spark.sql.warehouse.dir=s3a://foo/warehouse \
--conf spark.sql.extensions=org.apache.iceberg.spark.extensions.IcebergSparkSessionExtensions \
--conf spark.sql.catalog.my_catalog=org.apache.iceberg.spark.SparkCatalog \
--conf spark.sql.catalog.my_catalog.type=hadoop \
--conf spark.sql.catalog.my_catalog.warehouse=s3a://foo/iceberg-warehouse \
--conf spark.sql.catalog.my_catalog.io-impl=org.apache.iceberg.aws.s3.S3FileIO \
--conf spark.sql.catalog.my_catalog.catalog-impl=org.apache.iceberg.aws.glue.GlueCatalog \
--conf spark.sql.catalog.my_catalog.lock-impl=org.apache.iceberg.aws.glue.DynamoLockManager \
--conf spark.sql.catalog.my_catalog.lock.table=myGlueLockTable \
```

Need to add IAM policy like following (better replacing * with specific value to reduce permission scope):

```
        {
            "Effect": "Allow",
            "Action": "dynamodb:*",
            "Resource": "*"
        },
        {
            "Effect": "Allow",
            "Action": "glue:*",
            "Resource": "*"
        }
```

### Use Iceberg with JDBC Database

See https://github.com/apache/iceberg/blob/master/site/docs/jdbc.md

Spark config example:

```
--conf spark.jars=s3a://foo/iceberg-spark3-runtime-0.12.1.jar,s3a://foo/awssdk-url-connection-client-2.17.105.jar,s3a://foo/awssdk-bundle-2.17.105.jar,s3a://foo/mariadb-java-client-2.7.4.jar \
--conf spark.sql.warehouse.dir=s3a://foo/warehouse \
--conf spark.sql.extensions=org.apache.iceberg.spark.extensions.IcebergSparkSessionExtensions \
--conf spark.sql.catalog.my_catalog=org.apache.iceberg.spark.SparkCatalog \
--conf spark.sql.catalog.my_catalog.type=hadoop \
--conf spark.sql.catalog.my_catalog.warehouse=s3a://foo/iceberg-warehouse \
--conf spark.sql.catalog.my_catalog.io-impl=org.apache.iceberg.aws.s3.S3FileIO \
--conf spark.sql.catalog.my_catalog.catalog-impl=org.apache.iceberg.jdbc.JdbcCatalog \
--conf spark.sql.catalog.my_catalog.uri=jdbc:mysql://xxx.us-west-1.rds.amazonaws.com:3306/mydb \
--conf spark.sql.catalog.my_catalog.jdbc.verifyServerCertificate=false \
--conf spark.sql.catalog.my_catalog.jdbc.useSSL=true \
--conf spark.sql.catalog.my_catalog.jdbc.user=user1 \
--conf spark.sql.catalog.my_catalog.jdbc.password=xxx \
```

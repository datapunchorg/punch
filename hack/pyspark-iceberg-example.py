import pyspark
from pyspark.sql import SparkSession

if __name__ == "__main__":
  appName = "Write Table"
  print("Running: " + appName)

  spark = SparkSession.builder.appName(appName).getOrCreate()

  data = [("Apple",10), ("Banana",20), ("Pear",30)]
  columns = ["word","count"]
  df = spark.createDataFrame(data=data, schema = columns)
  df.printSchema()
  # df.show(truncate=False)
  df.createOrReplaceTempView("view_data")

  spark.sql("SHOW DATABASES").show()
  # spark.sql("CREATE DATABASE my_catalog.db01").show()
  spark.sql("CREATE TABLE IF NOT EXISTS my_catalog.db01.table01 (word string, count bigint) USING iceberg").show()
  spark.sql("INSERT INTO my_catalog.db01.table01 SELECT * FROM view_data").show()
  spark.sql("SELECT * FROM my_catalog.db01.table01").show()

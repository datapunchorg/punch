import pyspark
from pyspark.sql import SparkSession

if __name__ == "__main__":
  appName = "Sql Example"
  print("Running: " + appName)

  spark = SparkSession.builder.appName(appName).getOrCreate()
  # spark.sparkContext.setLogLevel("DEBUG")
  spark.sql("SELECT * FROM db_01.table_01").show()

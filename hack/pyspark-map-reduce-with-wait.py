import sys
import time
from pyspark import SparkContext, SparkConf

if __name__ == "__main__":
  appName = "Word Count - Python"
  print("Running: " + appName)

  mapSleepSeconds = 0
  reduceSleepSeconds = 0

  if len(sys.argv) >= 2:
    mapSleepSeconds = int(sys.argv[1])
  if len(sys.argv) >= 3:
    reduceSleepSeconds = int(sys.argv[2])

  conf = SparkConf().setAppName(appName)
  sc = SparkContext(conf=conf)

  words = sc.parallelize(["apple", "banana", "hello", "apple"])

  def map(word, sleep):
    print(f"map task sleeping {sleep} seconds")
    time.sleep(sleep)
    return (word, 1)

  def reduce(v1, v2, sleep):
    print(f"reduce task sleeping {sleep} seconds")
    time.sleep(sleep)
    return v1 + v2

  wordCounts = words.map(lambda word: map(word, mapSleepSeconds)).reduceByKey(lambda a,b: reduce(a, b, reduceSleepSeconds))

  wordCounts.foreach(lambda x: print(x))

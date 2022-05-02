
## Automated test for all Spark components

Please ensure [AWS CLI](https://aws.amazon.com/cli/) is installed and `aws s3` command working on your Mac.

```
make clean release
cd dist
./test/test-install.sh
./test/test-uninstall.sh
```

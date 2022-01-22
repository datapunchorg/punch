## Note

This project is not open sourced yet. We are still working on the progess to open source the project. Please not share the code publicly.


## How to create a package (on MacBook)

The following command will create a zip file for Punch.

```
make
```

Then check [User Guide](UserGuide.md) to see how to run punch command.

## TODO

1. Attach tag (e.g. punch-topology=xxx) to AWS resources created by punch
2. Mask password value in helm output (e.g. --set apiGateway.userPassword=xxx)
3. Allow patch topology like --patch foo.field1=value1
4. Allow set values by file like --values values.yaml
5. Support "sparkcli delete xxx"
6. Return HTTP 404 when sparkcli getting a non-existing application

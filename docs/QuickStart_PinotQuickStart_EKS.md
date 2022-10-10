Quick Start Guide: Use `punch` to create a Pinot Demo on AWS EKS
---

Short Version
---

Use following commands to run [Pinot Quick Start](https://github.com/apache/pinot/tree/master/kubernetes/helm/pinot) on AWS EKS:

```
make release
cd dist
./punch install PinotQuickStart --print-usage-example
```

Long Version
---

## Pre-requisite (One Time Setting)

### Install Homebrew

https://docs.brew.sh/Installation

```
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install.sh)"
```

### Install Helm

https://helm.sh/docs/intro/install/

```
brew update
brew install helm
```

### Install Kubectl

https://kubernetes.io/docs/tasks/tools/install-kubectl-macos/

```
brew install kubectl
```

### Set up AWS environment

1. Create AWS account, then download AWS Command Line Interface (https://aws.amazon.com/cli/).

2. Config AWS credential (https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-quickstart.html):

```
aws configure
```

### Get punch Distribution

Run `make release` in the root of this project, it will generate `dist` folder which contains `punch` binary.

Or download a pre-built zip file and unzip it:
```
curl -L -o dist.zip https://github.com/datapunchorg/punch/releases/download/0.9.0/dist.zip
unzip dist.zip
```

In the terminal, go the `dist` folder.

## Deploy Pinot and Realtime Ingestion Example

### Install PinotQuickStart

Run punch command:

```
./punch install PinotQuickStart --print-usage-example
```

The upper `punch install` will print out an AWS load balancer URL which you could browse to query data in Pinot, like following:

```
- step: deployPinotService
  output:
    pinotControllerUrl: http://a09c105e3207a49948d2ca2d5d5a7ee9-651003257.us-west-1.elb.amazonaws.com:9000
```

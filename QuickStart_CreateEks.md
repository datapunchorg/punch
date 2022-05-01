Note: this document only for running `punch` on Mac.

## Pre-requisite (One Time Setting)

### Install Homebrew

https://docs.brew.sh/Installation

```
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install.sh)"
```

### Install Helm

https://helm.sh/docs/intro/install/

```
brew install helm
```

### Install Go Programing Language

https://go.dev/doc/install

### Set up AWS environment

1. Create AWS account, then download AWS Command Line Interface (https://aws.amazon.com/cli/).

2. Config AWS credential (https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-quickstart.html):

```
aws configure
```

### Get punch Distribution

Run `make release` in the root of this project, it will generate `dist` folder which contains `punch` command.

In the terminal, go the `dist` folder.

## Provision EKS Cluster

### Run punch command to create a new EKS cluster

```
./punch install Eks
```

### Play with the EKS cluster

Now you could use [kubectl](https://kubernetes.io/docs/tasks/tools/) to explore the cluster.

```
aws eks update-kubeconfig --region us-west-1 --name my-eks-01
kubectl get nodes
kubectl get pods -A
```

### Run punch command to delete the EKS cluster

```
./punch uninstall Eks
```

---
title: Install 
weight: 1
---

### Pre-Installation

Liqo can be used with different topologies and scenarios. This impacts several installation parameters you will configure (e.g., API Server, Authentication).
Before installing Liqo, you should:
* Provision the clusters you would like to use with Liqo. If you need some advice about how to provision clusters on major providers, we have provided [here](./platforms/) some tips.
* Have a look to the [scenarios page](./pre-install) presents some common patterns used to expose and interconnect clusters.

### Quick Install

#### Pre-Requirements

To install Liqo, you have to install the following dependencies:

* [Helm 3](https://helm.sh/docs/intro/install/)

To install Liqo on your cluster, you should know:

* **PodCIDR**, the address space of IPs assigned to Pods
* **ServiceCIDR**:  the address space of IPs assigned to ClusterIP services

{{% notice note %}}
Liqo only supports Kubernetes >= 1.19.0.
{{% /notice %}}

Depending on the provider, you have different way to retrieve those parameters. For more information, you can check in the following subsections:

{{%expand "Kubeadm" %}}

To retrieve PodCIDR and ServiceCIDR in a Kubeadm cluster, you can just extract it from the kube-controller-manager spec:

```bash
POD_CIDR=$(kubectl get pods --namespace kube-system --selector component=kube-controller-manager --output jsonpath="{.items[*].spec.containers[*].command}" 2>/dev/null | grep -Po --max-count=1 "(?<=--cluster-cidr=)[0-9.\/]+")
SERVICE_CIDR=$(kubectl get pods --namespace kube-system --selector component=kube-controller-manager --output jsonpath="{.items[*].spec.containers[*].command}" 2>/dev/null | grep -Po --max-count=1 "(?<=--service-cluster-ip-range=)[0-9.\/]+")
echo "POD CIDR: $POD_CIDR"
echo "SERVICE CIDR: $SERVICE_CIDR"
```
{{% /expand%}}

{{%expand "AWS Elastic Kubernetes Service (EKS)" %}}
Retrieve information about your clusters, typing:
```bash
API_SERVER=$(aws eks describe-cluster --name __YOUR_CLUSTER_NAME --region __YOUR_CLUSTER_AWS_REGION__ | jq -r .cluster.endpoint | sed 's/https:\/\///g' ))
POD_CIDR=$(aws eks describe-cluster --name liqo-cluster --region eu-central-1 | jq -r '.cluster.resourcesVpcConfig.vpcId' | xargs aws ec2 describe-vpcs --vpc-ids --region eu-central-1 | jq '.Vpcs[0].CidrBlock')
```
{{% /expand%}}

{{%expand "Azure Kubernetes Service (AKS)" %}}

AKS clusters have by default with the following PodCIDR and ServiceCIDR:

If you are using Azure CNI:

```bash
SUBNET_ID=$(az aks list --query="[?name=='__YOUR_CLUSTER_NAME__']" | jq -r '.[0].agentPoolProfiles[0].vnetSubnetId')
POD_CIDR=$(az network vnet subnet show --ids ${SUBNET_ID} | jq -r .addressPrefix)
SERVICE_CIDR=$(az aks list --query="[?name=='__YOUR_CLUSTER_NAME__']" | jq -r ".[0].networkProfile.serviceCidr")
```

Or Kubenet:

```bash
POD_CIDR=$(az network vnet subnet show --ids ${SUBNET_ID} | jq -r ".[0].networkProfile.serviceCidr)
SERVICE_CIDR=$(az aks list --query="[?name=='__YOUR_CLUSTER_NAME__']" | jq -r ".[0].networkProfile.serviceCidr"
```


{{% /expand%}}
{{%expand "Google Kubernetes Engine (GKE)" %}}

```bash
SERVICE_CIDR=$(gcloud container clusters describe __YOUR_CLUSTER_NAME__ --zone -__YOUR_ZONE__ --project __YOUR_PROJECT_ID__ --format="json" | jq -r `.servicesIpv4Cidr`)
POD_CIDR=$(gcloud container clusters describe __YOUR_CLUSTER_NAME__ --zone -__YOUR_ZONE__ --project __YOUR_PROJECT_ID__ --format="json" | jq -r `.clusterIpv4Cidr`)
API_SERVER=$(gcloud container clusters describe __YOUR_CLUSTER_NAME__ --zone -__YOUR_ZONE__ --project __YOUR_PROJECT_ID__ --format="json" | jq -r `.endpoint`)
```

{{% /expand%}}
{{%expand "K3s" %}}

K3s clusters have by default with the following PodCIDR and ServiceCIDR:

| Variable               | Default | Description                                 |
| ---------------------- | ------- | ------------------------------------------- |
| `networkManager.config.podCIDR`             |    10.42.0.0/16     |
| `networkManager.config.serviceCIDR`         |    10.43.0.0/16     |
{{% /expand%}}

#### Set-Up Liqo Repository

Firstly, you should add the official Liqo repository to your Helm Configuration:

```bash
helm repo add liqo https://helm.liqo.io/
```

#### Set-up

The most important values you can set are the following:

| Variable               | Description                                 |
| ---------------------- | ------------------------------------------- |
| `networkManager.config.podCIDR`        | The cluster Pod CIDR                                 |
| `networkManager.config.serviceCIDR`    | The cluster Service CIDR                             |
| `discovery.config.clusterLabels`       | Labels used to characterize your cluster's resources |
| `auth.config.allowEmptyToken`          | Enable/disable [cluster pre-authentication](/User/Configure/Authentication)            |

Example:

```bash
helm install liqo liqo/liqo -n liqo --create-namespace  --set networkManager.config.podCIDR="10.42.0.0/16" --set networkManager.config.serviceCIDR="10.96.0.0/12" --set discovery.config.clusterLabels.region="A" --set discovery.config.clusterLabels.foo="bar" 
```

After a couple of minutes, the installation process will be completed. You can check if Liqo is running by:

```bash
kubectl get pods -n liqo
```

You should see a similar output:

```bash

```

#### Next Steps

After you have successfully installed Liqo, you may:

* [Configure](/user/configure): configure the Liqo security, the automatic discovery of new clusters and other system parameters.
* [Use](/user/use) Liqo: orchestrate your applications across multiple clusters.
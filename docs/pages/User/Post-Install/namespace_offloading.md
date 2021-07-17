---
title: Namespace offloading mechanism
weight: 3
---
* [Introduction](#introduction)
* [Parameters to be configured](#parameters-to-be-configured)
   * [Remote namespaces name](#1-remote-namespaces-name)
   * [Pod offloading strategy](#2-pod-offloading-strategy)
   * [Select remote clusters](#3-select-remote-clusters)
* [Default configuration](#default-configuration)
* [Custom configuration](#custom-configuration)
* [Comparison between Default configuration and Custom configuration](#comparison-between-default-configuration-and-custom-configuration)
* [Check the offloading status](#check-the-offloading-status)
   * [Offloading phase](#offloading-phase)
   * [Remote namespace conditions](#remote-namespace-conditions)
* [Offloading problems management](#offloading-problems-management)


## Introduction

The **namespace offloading mechanism** allows you to create a multicluster topology by selecting a set of clusters.
The granularity of this mechanism is at the namespace level, therefore it is necessary to create a local namespace
which will be replicated on all the selected remote clusters.

When you install Liqo on your cluster you can choose a set of labels that will characterize that cluster when exposed
remotely as a virtual node. As we will see later, choosing labels at installation time is essential if you want to
select only some clusters among those available after the peering phase.

By configuring some parameters you define constraints that will be imposed by the Liqo Webhook during the scheduling
of your workload. This approach allows you to define immutable control policies. Pods that do not respect these
imposed constraints remain pending.

##  Parameters to be configured

This section briefly explains which are the 3 parameters needed to configure the offloading. How to configure them
will be explained in a later section.

#### 1. Remote namespaces name.
These are the 2 possible values:

| Value               | Description |
| --------------      | ----------- |
| **EnforceSameName** | The remote namespaces have the same name as the namespace in the local cluster (this approach can lead to conflicts if a namespace with the same name already exists inside the selected remote clusters). |
| **DefaultName**     | The remote namespaces have the name of the local namespace followed by the local cluster-id to guarantee the absence of conflicts inside the remote clusters. |

> __NOTE__: The **DefaultName** value is recommended if you do not have particular constraints related to the remote
> namespaces name.

#### 2. Pod offloading strategy.
These are the 3 possible values:

| Value              | Description |
| --------------     | ----------- |
| **Local**          | The pods deployed in the local namespace are always scheduled inside the local cluster, never remotely.
| **Remote**         | The pods deployed in the local namespace are always scheduled inside the remote clusters selected, never locally.
| **LocalAndRemote** | The pods deployed in the local namespace can be scheduled both locally and remotely.

> __NOTE__: This strategy only applies to pods, services are always replicated inside the selected clusters.
#### 3. Select remote clusters.

The [k8s NodeAffinity](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#node-affinity) syntax
is used to select remote clusters thanks to the labels provided [at installation time](#introduction).
We will see an example to better understand how this selection occurs.

## Default configuration

You can choose the default configuration if the 3 parameters just presented are not relevant to your use case, while if
you are interested in specifying at least one of these you can skip this section and go directly to
[Custom configuration](#custom-configuration) section.

The parameters value for the default configuration are the following:

1. **DefaultName** for the remote namespaces, so ***localNamespaceName***+***localClusterID***.
2. **LocalAndRemote**, so pods deployed in the local namespace can be scheduled both locally and remotely.
3. All the remote clusters available after the peering phase are selected as target, so a remote namespace is created
   inside each remote cluster.

### Start offloading

Few simple steps are enough to have your environment ready:

1. First create the local namespace you want to replicate inside all the remote clusters.

```bash
  kubectl create namespace test-namespace
```

> __NOTE__: The namespace name cannot be longer than 63 characters according to the [RFC 1123](https://datatracker.ietf.org/doc/html/rfc1123).
> Since adding the cluster-id requires 37 characters, your namespace name can have at most 26 characters
2. Now add the label **liqo.io/enabled = true** to your namespace.

```bash
  kubectl label namespace test-namespace liqo.io/enabled=true
```

After these two simple steps, your work environment is ready, the remote namespaces with the default name should have
been created inside all the available remote clusters.

Here is an example with 3 remote clusters available:

![](/images/namespace-offloading/default-configuration.png)

You can start deploying your workload always keeping in mind that pods without additional constraints can be
scheduled both locally and remotely.

> __NOTE__: The pod offloading strategy **LocalAndRemote** does not impose constraints, it leaves the user the choice
> to deploy only locally or only remotely with the addition of further restrictions on the pod. While the **Remote** and
> **Local** strategies force the pods to be scheduled respectively only remotely and only locally,
> if the user tries to violate these constraints, adding conflicting restrictions, the pod will remain pending.
### Offloading termination

If you wish to terminate the offloading, simply remove the previously inserted label **liqo.io/enabled = true** or directly
delete the local namespace.
> __ATTENTION__: ending the offloading, all remote namespaces will be deleted with everything inside them.

## Custom configuration

If your use case requires particular configurations, you can manually set the 3 parameters as illustrated below:

### Start offloading

1. First create the local namespace you want to replicate inside the remote clusters
   (remember the name constraint, seen in the previous section).

```bash
  kubectl create namespace test-namespace
```

2. Now create a **NamespaceOffloading** resource inside the namespace (**NamespaceOffloading** is a Liqo CRD).
   A template of this resource is as follows:

{{% render-code file="static/examples/namespace-offloading-custom.yaml" language="yaml" %}}

As seen from the template the name is always **offloading**. A resource created with a different name will not activate
the remote namespaces creation.

#### Configure the 3 parameters inside the resource

Now let's see how to configure the 3 parameters in the **NamespaceOffloading Spec**:

1. **namespaceMappingStrategy**: it can assume the 2 values explained [in the parameter section](#1-remote-namespaces-name),
   other values are not accepted. If this parameter is omitted it will be set by default to the value of `DefaultName`.


2. **podOffloadingStrategy**:  it can assume the 3 values explained [in the parameter section](#2-pod-offloading-strategy),
   other values are not accepted. If this parameter is omitted it will be set by default to the value of `LocalAndRemote`.


3. **clusterSelector**: this field is used to specify the target remote clusters.
   If this parameter is omitted it will be set by default to the value:

```yaml
  clusterSelector:
    nodeSelectorTerms:
    - matchExpressions:
      - key: liqo.io/type
        operator: In
        values:
        - virtual-node
```
i.e. a remote namespace is created for each available remote cluster (all remote clusters expose the
label **liqo.io/type = virtual-node** by default).


Considering the **clusterSelector** of the template above:

We have 2 expressions in logical ***OR*** ( the 2 **matchExpressions** ), while all the
**keys** inside the matchExpression are in logical ***AND*** between them. So translating the expression we have:

> select all clusters that expose the label `region = A` **AND** `provider = Azure` **OR** clusters that
expose the label `region = B`
![](/images/namespace-offloading/custom-configuration.png)

> __ATTENTION__: The NamespaceOffloading resource must no longer be modified once created. If you want to change the
> offloading constraints you have to delete the resource and recreate it with a new configuration.

### Offloading termination

If you wish to terminate offloading, simply remove the previously created resource or directly delete the local namespace.

## Comparison between Default configuration and Custom configuration

The **Default configuration case** is equal to the **custom one** with the default value for the 3 parameters.
The Liqo enabling label addition creates in fact a default NamespaceOffloading resource like this:

{{% render-code file="static/examples/namespace-offloading-default.yaml" language="yaml" %}}

if you decide to use the label it is not possible to create a custom resource as there is already a namespace Offlaoding in that namespace

> __ATTENTION__: if you decide to use the label it is not possible to create a custom resource as there is already a
> **NamespaceOfflaoding** in that namespace. You have to remove the label first and then create the new resource.
## Check the offloading status

The NamespaceOffloading resource provides different information:

```bash
  kubectl get namespaceoffloading offloading -n test-namespace -o wide
```

The **namespaceMappingStrategy**, the **podOffloadingStrategy**, and the **remoteNamespaceName** must match your
configuration.

### Offloading phase

The global offloading status (**OffloadingPhase**) can assume different values:

| Value               | Description |
| --------------      | ----------- |
| **Ready**             |  Remote Namespaces have been correctly created inside previously selected clusters. |
| **NoClusterSelected** |  No cluster matches user constraints or constraints are not specified with the right syntax (in this second case an annotation is also set on the namespaceOffloading, specifying what is wrong with the syntax)        |
| **SomeFailed**        |  There was an error during creation of some remote namespaces. |
| **AllFailed**         |  There was an error during creation of all remote namespaces. |
| **Terminating**       |  Remote namespaces are undergoing graceful termination. |

### Remote namespace conditions

If you want more detailed information about the offloading status, you can check the **remoteNamespaceConditions**
inside the NamespaceOffloading resource:

```bash
   kubectl get namespaceoffloading offloading -n test-namespace -o yaml
```

The **remoteNamespaceConditions** field is a map which has as its key the ***remote cluster-id*** and as its value
a ***vector of conditions for the namespace*** created inside that remote cluster. There are two types of conditions:

1. **Ready**

   | Value   | Description |
      | ------- | ----------- |
   | **True**  |  The remote namespace is successfully created. |
   | **False** |  There was a problems during the remote namespace creation. |

2. **OffloadingRequired**

   | Value   | Description |
      | ------- | ----------- |
   | **True**  |  The creation of a remote namespace inside this cluster is required (the condition `OffloadingRequired = true` is removed when the remote namespace acquires a `Ready` condition). |
   | **False** |  The creation of a remote namespace inside this cluster is not required. |

> __NOTE__: The **RemoteNamespaceCondition** syntax is the same of the standard [NamespaceCondition](https://pkg.go.dev/k8s.io/api/core/v1@v0.21.0#NamespaceCondition).
## Offloading problems management

The NamespaceOffloading resource reports possible problems through the remote namespace condition and the global
offloading status. The implemented retry mechanism should resolve transient errors, however if the problem persists,
check that the namespace creation request is valid (e.g. length of the namespace name is less than 27 characters
as seen above) or that a namespace with the same name does not already exist inside the remote cluster.
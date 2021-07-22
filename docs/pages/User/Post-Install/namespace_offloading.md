---
title: Namespace offloading mechanism
weight: 3
---
##TODO: Index



## Introduction

The ***namespace offloading mechanism*** allows you to create a multicluster topology by selecting a set of clusters.
Once you have created your topology you have the possibility to enforce some very strict constraints on where
your applications need to be deployed and how these can be reached from the various clusters. It's not just about
exploiting the resources of multiple clusters but to build a real architecture suited to your needs.


## Configure cluster labels at Liqo installation time

When you install Liqo on your cluster you can choose a set of labels that will characterize it when exposed
remotely. If you want to select this cluster among the others you will have to refer to these labels. Therefore,
although it is possible to choose any value for the labels, it would be advisable to assign values as significant
as possible. If you want to see how to define these labels look at the
[Getting started section](#offloading-problems-management).


> __NOTE__: If you don't need to select specific clusters, you are not required to add labels at Liqo installation time.
> As we will see later if no selection filter is specified, the Liqo mechanism will automatically select all available
> remote clusters to offload your workload.


## How namespace replication works

The granularity of the offloading mechanism is at the namespace level. Therefore it is necessary to create a local
namespace also called "***Liqo namespace***". A resource containing the offloading configuration is created within
this namespace, this resource is a Liqo CRD called **NamespaceOffloading**. More details about this resource will
be described in the following section.

> __ATTENTION__: In ***Liqo 0.3*** the NamespaceOffloading must contain the configuration at creation time,
> once created it must ***no longer be modified***. If you want to change the offloading constraints you have
> to delete the resource and recreate it with a new configuration.

Once the resource has been generated some Liqo controllers will replicate the local namespace inside all the remote
clusters selected through the resource configuration. This configuration allows you to impose hard constraints
that will be respected for the whole offloading duration.

The offloading status can be monitored directly from the NamespaceOffloading resource as we will see later.

## NamespaceOffloading structure and how to configure it

A template of the NamespaceOffloading resource is as follows:

{{% render-code file="static/examples/namespace-offloading-default.yaml" language="yaml" %}}

The name of the resource must be always "**offloading**" to ensure the uniqueness of a single configuration for
that namespace. A resource created with a different name will not trigger the remote namespaces creation.

Now let's see how to configure the parameters in the ***NamespaceOffloading Spec***:

1. The **namespaceMappingStrategy** can assume 2 values:

   | Value               | Description |
      | --------------      | ----------- |
   | **EnforceSameName** | The remote namespaces have the same name as the namespace in the local cluster (this approach can lead to conflicts if a namespace with the same name already exists inside the selected remote clusters). |
   | **DefaultName**     | The remote namespaces have the name of the local namespace followed by the local cluster-id to guarantee the absence of conflicts inside the remote clusters. |

   Other values are not accepted. If this field is omitted it will be set by default to the value of **DefaultName**.

   > __NOTE__: The **DefaultName** value is recommended if you do not have particular constraints related to the remote
   > namespaces name.


2. The **podOffloadingStrategy** can assume the 3 values:

   | Value              | Description |
      | --------------     | ----------- |
   | **Local**          | The pods deployed in the local namespace are always scheduled inside the local cluster, never remotely.
   | **Remote**         | The pods deployed in the local namespace are always scheduled inside the remote clusters, never locally.
   | **LocalAndRemote** | The pods deployed in the local namespace can be scheduled both locally and remotely.
   Other values are not accepted. If this parameter is omitted it will be set by default to the value of **LocalAndRemote**.

   The pod offloading strategy **LocalAndRemote** does not impose constraints, it leaves the scheduler the choice
   to deploy locally or remotely. While the **Remote** and **Local** strategies force the pods to be scheduled respectively only remotely and only locally.
   If the user tries to violate these constraints, adding conflicting restrictions, the pod will remain pending.

   If you select 2 remote clusters and a podOffloadingStrategy ***LocalAndRemote***, you don't know which cluster the
   pod will be scheduled inside. The scheduler may also decide to keep the pod locally based on available resources. if
   you want to force a pod to be scheduled on a specific cluster you have to add further restrictions on it.

   > __NOTE__: This strategy only applies to pods, services are always replicated inside all the selected clusters.

3. The **clusterSelector** field is used to specify the target remote clusters with the
   [k8s NodeAffinity](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#node-affinity) syntax,
   it uses the labels provided at installation time by every cluster.
   > __NOTE__: This selector will be imposed on ***every pod*** scheduled inside the Liqo namespace, therefore you have the
   > guarantee that your pods can only be scheduled on the clusters you have selected.

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
   i.e. all available remote cluster are selected as possible target (each remote cluster exposes the
   **liqo.io/type = virtual-node** label by default).

## Reference topology

Let's consider a reference topology with 3 clusters which is also used in the getting started section
(todo: #reference it in the getting started)

We can observe the 3 clusters with Liqo already installed. Each one has its own labels:

// image 1: 3 clusters with liqo installed and labels

Let's imagine that cluster 1 has a [peering](#) with both cluster 2 and cluster 3. During the peering,
the labels chosen at installation time by the cluster 2 and the cluster 3 become known to cluster 1 which can use them
to select one cluster rather than the other.

The situation is well represented by the image below:

// image 2: cluster 1 has the labels of cluster 2 and 3


## The two offloading scenarios

If your goal is to use the resources of multiple clusters to deploy your workload regardless of
cluster type and how your pods are scheduled, then the **Default configuration** can be great
for your use case.

If, on the other hand, your scenario is more complex and you need a particular configuration (e.g. pods scheduled only on some
clusters, locally secured pods with only services that remotely expose them, have remote namespaces with the same name
to activate a service mesh) then you have to use a **Custom configuration**.

Let's see what is the difference between the two configurations just mentioned.


## Default configuration

Let's consider our reference topology. On cluster 1 we want to deploy an application that is also able to exploit
the resources of the other two clusters. It doesn't matter where the pods are scheduled or what is the remote namespaces
name, the important thing is that the application is as efficient as possible. In this case 2 simple steps are enough:

### Start offloading

1. Create the local namespace inside the cluster 1.

```bash
  kubectl create namespace test-namespace
```

> #### Name constraints:
>The namespace name cannot be longer than 63 characters according to the [RFC 1123](https://datatracker.ietf.org/doc/html/rfc1123).
> Since adding the cluster-id requires 37 characters, your namespace name can have at most 26 characters


2. Now add the label **liqo.io/enabled = true** to your namespace.

```bash
  kubectl label namespace test-namespace liqo.io/enabled=true
```

Your work environment is ready, you are able to deploy your apps also inside remote clusters.

When the label is inserted, a default NamespaceOffloading is automatically created. The resource is exactly equal to the
template seen above, the parameters have all the default values:

1. **namespaceMappingStrategy** = **DefaultName**.
2. **podOffloadingStrategy** = **LocalAndRemote**.
3. **clusterSelector** = the default one previosly seen.

A remote namespace with ***DefaultName*** is created inside all available remote clusters (2 and 3), and the pods
can be scheduled both locally and remotely. The current situation is represented by the following figure, which shows
how some pods have been deployed locally while others inside the cluster 2. The scheduler is not required to schedule pods
inside all previously selected remote clusters. In this case the cluster 3, although selected, does not receive any pods.

![](/images/namespace-offloading/default-configuration.png)

> __ATTENTION__: if you decide to use the label it is not possible to create a custom resource as there is already a
> **NamespaceOfflaoding** in that namespace. You have to remove the label first and then create the new resource.


### Offloading termination

To terminate the offloading, simply remove the previously inserted label **liqo.io/enabled = true** or directly
delete the resource inside the namespace.
> __ATTENTION__: ending the offloading, all remote namespaces will be deleted with everything inside them.


## Custom configuration

> __NOTE__ : Liqo 0.2.1 allowed you to use only the label mechanism. In Liqo 0.3
> custom configuration is introduced and so the possibility to configure offloading.

Let's consider our reference topology. We want to deploy an application inside the local namespace (cluster 1). It can
stay locally or be offloaded only inside cluster 2 because both clusters respect particular security policies
necessary for our application.

### Start offloading

1. Create the local namespace inside the cluster 1. (remember the name constraint, seen in the previous section #reference).

```bash
  kubectl create namespace test-namespace
```

2. Now create a **NamespaceOffloading** resource inside the namespace. Consider a resource like this:

{{% render-code file="static/examples/namespace-offloading-custom.yaml" language="yaml" %}}

As we recall from our reference topology, cluster 2 was installed with exactly the required labels
from our clusterSelector. Only one remote namespace with the same name as the local one (** EnforceSameName **) will be created
inside the cluster 2. The pods can be scheduled both locally and inside the cluster 2. The situation described in the
figure below shows how the scheduler decided to schedule all pods locally in this case:

![](/images/namespace-offloading/custom-configuration.png)

> nota: se si dovesse fare il peering con un nuovo cluster che metcha i vincoli del selector allora un namespace remoto
> verrebbbe creato anche su quel cluster e anche lui diventerebbe target per possibili offloading.

### Offloading termination

If you wish to terminate offloading, simply remove the previously created resource or directly delete the local namespace.


## Dynamic peering

// Todo:

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

## Offloading problems management (#TODO: move it to the troubleshooting section)

The NamespaceOffloading resource reports possible problems through the remote namespace condition and the global
offloading status. The implemented retry mechanism should resolve transient errors, however if the problem persists,
check that the namespace creation request is valid (e.g. length of the namespace name is less than 27 characters
as seen above) or that a namespace with the same name does not already exist inside the remote cluster.
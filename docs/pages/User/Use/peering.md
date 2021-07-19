---
title: Add a remote cluster 
weight: 1
---

## Overview

In Liqo, peering establishes an administrative connection between two clusters and enables the resource sharing across them.
It is worth noticing that peering is uni-directional. This implies that resources can be shared only from a cluster to another and not the vice-versa. Obviously, it can be optionally be enabled bi-directionally, enabling a two-way resource sharing.

### Peer with a new cluster

To peer with a new cluster, you have to create a ForeignCluster CR.

#### Add a new ForeignCluster 

A `ForeignCluster` resource represents a cluster inside the Liqo ecosystems. It specifies the parameters to be contacted by the Liqo components. More precisely, it requires the authentication service URL and the port to be set: it is the backend of the
authentication endpoint, mandatory to peer with another Liqo cluster.


{{%expand "To retrieve the Liqo authentication endpoint on your cluster" %}}

You can just type:

```bash
kubectl get svc -n liqo liqo-auth
```

You will obtain an output similar to the following:

```bash
NAME           TYPE       CLUSTER-IP    EXTERNAL-IP        PORT(S)        AGE
liqo-auth   LoadBalancer 10.81.20.99   100.200.100.200   443:30740/TCP   2m7s
```

From another cluster, you may use the service External-IP to reach the Authentication service (e.g.; 100.200.100.200:443 in the example). Alternatively, if you have direct connectivity with the cluster nodes, you can use the corresponding NodePort with one of the cluster node IPs.
{{% /expand%}}

An example of `ForeignCluster` resource can be:

```yaml
apiVersion: discovery.liqo.io/v1alpha1
kind: ForeignCluster
metadata:
  name: my-cluster
spec:
  join: true # optional (defaults to false)
  authUrl: "https://<ADDRESS>:<PORT>"
```

When you create the ForeignCluster, the Liqo control plane will contact the `authURL` (i.e. the public URL of a cluster
authentication server) to retrieve all the required cluster information.

#### Peering

If you created the ForeignCluster with the peering disabled (i.e. `join` set to false), you can enable the peering by setting its join flag as follows:

```bash
kubectl patch foreignclusters "$foreignClusterName" \
  --patch '{"spec":{"join":true}}' \
  --type 'merge'
```

To disable the peering, it is enough to patch the `ForeignCluster` resource as follows:

```bash
kubectl patch foreignclusters "$foreignClusterName" \
  --patch '{"spec":{"join":false}}' \
  --type 'merge'
```

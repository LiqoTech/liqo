---
title: Demo/Tutorial with KubernetesOnDesktop
weight: 4
---

## About this demo/tutorial

In this demo/tutorial we will show you how to install [liqo.io](https://liqo.io) on two [k3s](https://k3s.io/) cluster from scratch and then run a real application ([KubernetesOnDesktop](#about-kubernetesondesktop)) by using the foreign cluster.

N.B.:
From now on, when we'll talk about "*local cluster*" we'll refer to the one that will run the `cloudify` script ([see afterwards](#about-kubernetesondesktop)), and when we'll talk about "*foreign cluster*" we'll refer to the other one.

## About KubernetesOnDesktop

[KubernetesOnDesktop](https://github.com/netgroup-polito/KubernetesOnDesktop) is a University project with the aim of developing a cloud infrastructure to run user application in a remote cluster node. 
It uses a client/server VNC+PulseAudio+SSH infrastructure that schedules the application `pod` in a k8s remote node and redirects the GUI (through VNC) and the sound (through PulseAudio+SSH) in a second `pod` scheduled in the node in which the `cloudify` application is running. For further information see [KubernetesOnDesktop](https://github.com/netgroup-polito/KubernetesOnDesktop) GitHub page.

So far, the supported applications are firefox, libreoffice and blender (with the ability to use NVIDIA CUDA driver if the node which the `pod` will be executed in has a NVIDIA graphic card). Anyway, thanks to a huge use of templates, it is possible to scale up to many more applications.

This project uses several kubernetes components, such as `deployments`, `jobs`, `services` (with `ClusterIP` and `NodePort` `type` values) and `secrets`. All those components, as you can see afterwards, are supported by [liqo.io](https://liqo.io).

When executed, through `cloudify` command, the application will create:
* a `secret` containing a ssh key;
* a `deployment` containing the application (e.g. blender) and the VNC server, whose `pod` will be scheduled on a remote node with respect to the node `cloudify` is launched from;
* a `service` of `type` `NodePort` (that automatically creates a `ClusterIP` `type` too, as you can see in [k8s official documentation](https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services-service-types)) that makes the `pod` created from the `deployment` above reachable from other `pod`s in the cluster and from the outside;
* a `job` executing the VNC viewer, whose `pod` will be scheduled in the same node `cloudify` is launched from.

## Demo goal

In this demo, we will try to execute a blender `pod` in a *foreign cluster* (that is represented in the *local cluster* as a *virtual node* named `liqo-<...>`) and a viewer `pod` in the *local cluster*.

Thanks to the *foreign cluster* virtualization as a *local cluster* node, the `cloudify` application will automatically schedule the `pod`s as described above and will use the [K8s DNS for services](https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/) for the communications between the `pod`s. In fact, even if there are two separated clusters and the `pod`s will be scheduled one for each, it's not required to use the `NodePort` because the *foreign cluster* is actually a *virtual node* of the *local cluster*. So, to reach the `pod` scheduled in the *foreign cluster* from the one scheduled in the *local cluster*, [`ServiceURL:Port`](https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/#services) will be used instead of `NodeIP:NodePort`.

## Installation of the required software
To install all the required software we need to follow this steps:

1. [Install k3s](#install-k3s) in both clusters;
2. [Install liqo.io](#install-liqo-io) in both clusters; 
3. [Install KubernetesOnDesktop](#install-kubernetesondesktop) just in one of the clusters (the so called *local cluster*).

### Install k3s
Assuming you already have two linux (we tested it with Ubuntu Desktop 20.04 LTS) machine or Virtual Machine up and running in the same subnet, we can install [k3s](https://k3s.io/) by using the official script as you can see in [K3s Quick-Start Guide](https://rancher.com/docs/k3s/latest/en/quick-start/). So, you only need to run the following:

```bash
curl -sfL https://get.k3s.io | sh -
```

When the script ends, to make [liqo.io](https://liqo.io) work properly, you need to modify the `/etc/systemd/system/k3s.service` file by adding the `--kube-apiserver-arg anonymous-auth=true` service execution parameter. After this operation, your `k3s.service` file should be like this:

```
[Unit]
Description=Lightweight Kubernetes
Documentation=https://k3s.io
Wants=network-online.target

[Install]
WantedBy=multi-user.target

[Service]
Type=notify
EnvironmentFile=/etc/systemd/system/k3s.service.env
KillMode=process
Delegate=yes
# Having non-zero Limit*s causes performance problems due to accounting overhead
# in the kernel. We recommend using cgroups to do container-local accounting.
LimitNOFILE=1048576
LimitNPROC=infinity
LimitCORE=infinity
TasksMax=infinity
TimeoutStartSec=0
Restart=always
RestartSec=5s
ExecStartPre=-/sbin/modprobe br_netfilter
ExecStartPre=-/sbin/modprobe overlay
ExecStart=/usr/local/bin/k3s \
server \
--kube-apiserver-arg anonymous-auth=true \

```

Now you need to apply the changes by executing the following:

```bash
systemctl daemon-reload
systemctl restart k3s.service
```

Finally, to use `kubectl` command to interact with the cluster without `sudo`, you need to give the `k3s.yaml` config file the required privileges and to copy it in `~/.kube/config` by doing the following:

```bash
sudo chmod 666 /etc/rancher/k3s/k3s.yaml
cp /etc/rancher/k3s/k3s.yaml ~/.kube/config
```

Before proceeding with the [liqo.io](https://liqo.io) installation, wait for all the pod to be in `Running` status. You can check it by executing `kubectl get pod --all-namespaces`.

### Install liqo.io
To install [liqo.io](https://liqo.io) you have to set the required environment variables and use the script provided in the project by doing the following:

```bash
export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
export POD_CIDR=10.32.0.0/16
export SERVICE_CIDR=10.10.0.0/16
curl https://raw.githubusercontent.com/LiqoTech/liqo/master/install.sh | bash
```

For further information see the [Liqo Installation Guide](/user/gettingstarted/install/#custom-install).

Before proceding with the installation of [KubernetesOnDesktop](https://github.com/netgroup-polito/KubernetesOnDesktop) in one of the two clusters, wait for all the `pod`s in `liqo.io` `namespace` to be up and running in both clusters. You can check it by executing `kubectl get pod -n liqo.io` in both clusters.

Due to the fact that both (virtual) machines share the same subnet, each liqo cluster will automatically join the foreign one! See the liqo [Discovery](/user/configure/discovery/) and [Peering](/user/gettingstarted/peer/) documentation.

### Install KubernetesOnDesktop
Now that both [k3s](https://k3s.io/) and [liqo.io](https://liqo.io) are up and running, we can install [KubernetesOnDesktop](https://github.com/netgroup-polito/KubernetesOnDesktop) by cloning the git repository and launching the install.sh script as follows:

```bash
git clone https://github.com/netgroup-polito/KubernetesOnDesktop.git
cd KubernetesOnDesktop
sudo ./install.sh
```

Now we are ready to run the `cloudify` script.

## Run the KubernetesOnDesktop demo
To run the demo we need to execute the `cloudify` command. We can do it as follows:

```bash
cloudify -t 500 -r 2 -e blender
```
Parameters meaning:
* -t 500 -> specifies a timeout. If the pods doesn't have the `Running` status before the timeout the native application will be run (if any). It's strongly recommended to specify a huge value for this parameter the very first time you execute the application, this is because a lot of time will be spent to pull the application and the viewer images;
* -r 2 -> specifies the run mode. In this case the viewer will be a k8s `pod` too (as the application one) and will be scheduled on the current node;
* -e -> enable tunnel encryption between the app `pod` and the viewer `pod`;
* blender -> the (supported) application we want to execute. If you have a NVIDIA graphic card (with the required drivers already installed as specified in the [NVIDIA Quickstart](https://github.com/NVIDIA/nvidia-docker#quickstart)) in the node the `pod` will be executed in, you can use that card with blender!!
If you need help about the execution parameters, please run `cloudify -h`.

The `cloudify` application will create the `k8s-on-desktop` `namespace` (if not present) and will apply on it the `liqo.io/enabled=true` `label` so that this `namespace` could be reflected to the liqo.io *foreign cluster*.

Also, a label to the local `node` will be applied to let k3s scheduling the pods according to the node affinity specified in the `kubernetes/deployment.yaml`. This rule specifies that the application `pod` must run in a node that is not the local one. In reverse, inside `kubernetes/vncviewer.yaml` it is specified that the viewer must be executed on the local node.

### Check the created resources and where the pods run
When the GUI appears on the machine running the `cloudify` script, you can check the created resources by running on it the following:
```bash
kubectl get deployment -n k8s-on-desktop    #This will show you the application deployment (blender in this example)
kubectl get jobs -n k8s-on-desktop          #This will show you the vncviewer job
kubectl get secrets -n k8s-on-desktop       #This will show you the secret containing the ssh key
kubectl get pod -n k8s-on-desktop -o wide   #This will show you the running pods and which node the were scheduled in
```

Actually you can execute the commands above in both the clusters but pay attention to the `namespace`! In fact, in the *foreign cluster* the `k8s-on-desktop` `namespace` will be reflected by adding a suffix as follows `k8s-on-desktop-<...>`. So, to retrieve that `namespace`, execute the following in the *foreign cluster*:
```bash
kubectl get namespaces
```

Now, you can execute on the *foreign cluster* all the `kubectl` listed above by replacing the `namespace` with the one obtained with the previous command.
In this case, you will see that only the `secret`, the `deployment` and the application `pod` (in this example blender) will exist in this cluster. This is because the other resources (related to vncviewer) will be only in the *local cluster*.

## Cleanup KubernetesOnDesktop installation
To clean up the KubernetesOnDesktop installation you need to do the following:

1. Execute uninstall.sh script:
```bash
cd /path/to/KubernetesOnDesktop #The path to the KubernetesOnDesktop git folder
sudo ./uninstall.sh
```
2. Delete the `k8s-on-desktop` `namespace`:
```bash
kubectl delete namespace k8s-on-desktop
```
3. Delete the KubernetesOnDesktop git folder. Assuming you are inside the folder, execute:
```bash
cd ..
rm -fr KubernetesOnDesktop
```

## Teardown k3s and liqo.io
To teardown k3s and liqo.io just run the following in both the nodes:
```bash
k3s-uninstall.sh
```
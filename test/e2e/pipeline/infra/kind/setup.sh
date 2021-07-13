#!/bin/bash
CLUSTER_NAME=cluster

#export CLUSTER_NUMBER=3
#export K8S_VERSION=v1.19
export DISABLE_KINDNET=false
# Export KIND path to the rest of the pipeline
KIND="${BINDIR}/kind"

if [[ ${CNI} != "kindnet" ]]; then
  export DISABLE_KINDNET=true
fi

export SERVICE_CIDR=10.100.0.0/16
export POD_CIDR=10.200.0.0/16

for i in $(seq 1 "${CLUSTER_NUMBER}");
do
   if [[ ${POD_CIDR_OVERLAPPING} != "true" ]]; then
       export POD_CIDR=10.${i}.0.0/16
   fi
   envsubst < "${TEMPLATE_DIR}/templates/cluster-templates.yaml.tmpl" > "${TMPDIR}/liqo-cluster-${CLUSTER_NAME}${i}.yaml"
   echo "Creating cluster ${CLUSTER_NAME}${i}..."
   ${KIND} create cluster --name "${CLUSTER_NAME}${i}" --kubeconfig "${TMPDIR}/kubeconfigs/liqo_kubeconf_${i}" --config "${TMPDIR}/liqo-cluster-${CLUSTER_NAME}${i}.yaml" --wait 2m
done
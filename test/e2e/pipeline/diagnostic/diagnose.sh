for i in $(seq 1 ${CLUSTER_NUMBER});
do
   export KUBECONFIG=${TMPDIR}/kubeconfigs/liqo_kubeconf_${i}
   kubectl get po -A
   kubectl get all -n liqo
   kubectl get crd -A
done;
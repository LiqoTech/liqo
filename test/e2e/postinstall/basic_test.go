package postinstall

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"github.com/liqotech/liqo/test/e2e/testutils/tester"
	"github.com/liqotech/liqo/test/e2e/testutils/util"
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Liqo E2E Suite")
}

var _ = Describe("Liqo E2E", func() {
	var (
		ctx         = context.Background()
		testContext = tester.GetTester(ctx)
		namespace   = "liqo"
		interval    = 3 * time.Second
		timeout     = 5 * time.Minute
	)

	Describe("Assert that Liqo is up, pod offloading and network connectivity are working", func() {
		Context("Check Join Status", func() {

			DescribeTable("Liqo pods are up and running",
				func(cluster tester.ClusterContext, namespace string) {
					readyPods, notReadyPods, err := util.ArePodsUp(ctx, cluster.Client, testContext.Namespace)
					Eventually(func() bool {
						return err == nil
					}, timeout, interval).Should(BeTrue())
					Expect(len(notReadyPods)).To(Equal(0))
					Expect(len(readyPods)).Should(BeNumerically(">", 0))
				},
				Entry("Pods UP on cluster 1", testContext.Clusters[0], namespace),
				Entry("Pods UP on cluster 2", testContext.Clusters[1], namespace),
			)

			DescribeTable("Liqo Virtual Nodes are ready",
				func(homeCluster tester.ClusterContext, namespace string) {
					nodeReady := util.CheckVirtualNodes(ctx, homeCluster.Client)
					Expect(nodeReady).To(BeTrue())
				},
				Entry("VirtualNode is Ready on cluster 2", testContext.Clusters[0], namespace),
				Entry("VirtualNode is Ready on cluster 1", testContext.Clusters[1], namespace),
			)
		})
	})
})
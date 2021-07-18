package deployment_e2e

import (
	"context"
	"testing"
	"time"

	"github.com/liqotech/liqo/test/e2e/testutils/tester"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Liqo E2E Suite")
}

var _ = Describe("Microservice application deployment across three clusters", func() {
	/*
		These End to End tests check the support for applications
		spanning on more than 2 clusters is correct and complete.
		Several Liqo components are involved in this new feature:
			- Virtual Kubelet, responsible of service (and therefore endpoints) reflection.
			- IPAM, asked to map endpoint IPs by the VK via gRPCs
			- Gateway, that reconciles on a per-cluster resource called NatMapping that is
			  updated by the IPAM whenever a new mapping with ExternalCIDR is carried out.
			- The iptables Liqo driver, responsible of insertion/deletion of NAT rules to
			   ensure communication between clusters. It is consumed by the Gateway for
			   inserting DNAT rules used to redirect traffic towards endpoints.

		The following tests will deploy three clusters A, B and C; make peering
		sessions between A/B and B/C; deploy a microservice application on B;
		cluster B will have two virtual nodes corresponding to foreign clusters
		A and C, therefore we expect some Pods will be scheduled on those clusters.
		We'll test the deployed application, making HTTP requests, in order to guarrantee
		everything work as expected.
	*/
	var (
		ctx         = context.Background()
		testContext = tester.GetTester(ctx)
		namespace   = "liqo"
		interval    = 3 * time.Second
		timeout     = 5 * time.Minute
	)
})

package utils_test

import (
	"github.com/vishvananda/netlink"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestUtils(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Utils Suite")
}

var _ = BeforeSuite(func() {
	// Create dummy link.
	err := netlink.LinkAdd(&netlink.Dummy{LinkAttrs: netlink.LinkAttrs{Name: interfaceName}})
	Expect(err).ShouldNot(HaveOccurred())
})
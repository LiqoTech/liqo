package routeoperator

import (
	"net"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vishvananda/netlink"

	"github.com/liqotech/liqo/pkg/liqonet/overlay"
)

var (
	vxlanConfig = &overlay.VxlanDeviceAttrs{
		Vni:      1800,
		Name:     "vxlan.route",
		VtepPort: 4789,
		VtepAddr: nil,
		Mtu:      1450,
	}
	overlayDevice overlay.VxlanDevice
)

func TestRouteOperator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "RouteOperator Suite")
}

var _ = BeforeSuite(func() {
	/*** OverlayOperator configuration ***/
	link, err := setUpVxlanLink(vxlanConfig)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(link).ShouldNot(BeNil())
	overlayDevice.Link = link.(*netlink.Vxlan)
	// Configure existing neigh.
	peerIP := net.ParseIP(overlayPeerIP)
	Expect(peerIP).NotTo(BeNil())
	peerMAC, err := net.ParseMAC(overlayPeerMAC)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(peerMAC).NotTo(BeNil())
	overlayExistingNeigh.IP = peerIP
	overlayExistingNeigh.MAC = peerMAC
	// Configure neigh.
	peerIP1 := net.ParseIP(overlayPodIP)
	Expect(peerIP).NotTo(BeNil())
	peerMAC, err = net.ParseMAC(overlayAnnValue)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(peerMAC).NotTo(BeNil())
	overlayNeigh.IP = peerIP1
	overlayNeigh.MAC = peerMAC
	// Setup envtest
	Expect(setupOverlayTestEnv()).To(BeNil())
})

var _ = AfterSuite(func() {
	Expect(netlink.LinkDel(overlayDevice.Link)).ShouldNot(HaveOccurred())
})

func setUpVxlanLink(attrs *overlay.VxlanDeviceAttrs) (netlink.Link, error) {
	err := netlink.LinkAdd(&netlink.Vxlan{
		LinkAttrs: netlink.LinkAttrs{
			Name:  attrs.Name,
			MTU:   attrs.Mtu,
			Flags: net.FlagUp,
		},
		VxlanId:  attrs.Vni,
		SrcAddr:  attrs.VtepAddr,
		Port:     attrs.VtepPort,
		Learning: true,
	})
	if err != nil {
		return nil, err
	}

	link, err := netlink.LinkByName(attrs.Name)
	if err != nil {
		return nil, err
	}
	return link, nil
}

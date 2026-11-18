package v1

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	ifaces "phenix/types/interfaces"
)

func TestTopologyInitHostnames(t *testing.T) {
	// minimal node that survives setDefaults (which dereferences HardwareF).
	node := func(hostname string) *Node {
		return &Node{
			GeneralF:  &General{HostnameF: hostname},
			HardwareF: &Hardware{},
		}
	}

	t.Run("rejects duplicate hostnames", func(t *testing.T) {
		topo := &TopologySpec{
			NodesF: []*Node{node("dup"), node("other"), node("dup")},
		}

		err := topo.Init("phenix")
		if err == nil {
			t.Fatalf("expected an error, got nil")
		}

		if !strings.Contains(err.Error(), `duplicate node hostname "dup"`) {
			t.Fatalf("error %q does not name the duplicate hostname", err.Error())
		}
	})

	t.Run("rejects hostnames differing only in case", func(t *testing.T) {
		topo := &TopologySpec{
			NodesF: []*Node{node("ADServer"), node("adserver")},
		}

		err := topo.Init("phenix")
		if err == nil {
			t.Fatalf("expected an error, got nil")
		}

		if !strings.Contains(err.Error(), `duplicate node hostname "adserver"`) {
			t.Fatalf("error %q does not name the duplicate hostname", err.Error())
		}
	})

	t.Run("allows unique hostnames", func(t *testing.T) {
		topo := &TopologySpec{
			NodesF: []*Node{node("a"), node("b"), node("c")},
		}

		if err := topo.Init("phenix"); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})
}

func TestTopologyNetworkSegments(t *testing.T) {
	external := true
	topology := &TopologySpec{NodesF: []*Node{
		{
			GeneralF: &General{HostnameF: "vm1"},
			NetworkF: &Network{InterfacesF: []*Interface{
				{NameF: "eth0", TypeF: "ethernet", ProtoF: "static", VLANF: "control", AddressF: "10.0.0.2", MaskF: 24, MACF: "00:00:00:00:00:01", MTUF: 1500},
				{NameF: "eth1", BridgeF: "plant", VLANF: "ics"},
			}},
		},
		{
			GeneralF: &General{HostnameF: "vm2"},
			NetworkF: &Network{InterfacesF: []*Interface{
				{NameF: "eth0", BridgeF: "phenix", VLANF: "control"},
			}},
		},
		{
			GeneralF:  &General{HostnameF: "hil"},
			ExternalF: &external,
			NetworkF: &Network{InterfacesF: []*Interface{
				{NameF: "eth0", BridgeF: "plant", VLANF: "ics"},
			}},
		},
		{
			GeneralF: &General{HostnameF: "empty"},
			NetworkF: &Network{InterfacesF: []*Interface{nil}},
		},
		nil,
	}}

	want := []ifaces.NetworkSegment{
		{
			Bridge: "phenix",
			VLAN:   "control",
			Endpoints: []ifaces.NetworkSegmentEndpoint{
				{Hostname: "vm1", Interface: "eth0", Type: "ethernet", Proto: "static", Address: "10.0.0.2", Mask: 24, MAC: "00:00:00:00:00:01", MTU: 1500, External: false},
				{Hostname: "vm2", Interface: "eth0", External: false},
			},
		},
		{
			Bridge: "plant",
			VLAN:   "ics",
			Endpoints: []ifaces.NetworkSegmentEndpoint{
				{Hostname: "vm1", Interface: "eth1", External: false},
				{Hostname: "hil", Interface: "eth0", External: true},
			},
		},
	}

	if diff := cmp.Diff(want, topology.NetworkSegments()); diff != "" {
		t.Errorf("TopologySpec.NetworkSegments() mismatch (-want +got):\n%s", diff)
	}
}

func TestTopologyNetworkSegmentsNil(t *testing.T) {
	var topology *TopologySpec

	if got := topology.NetworkSegments(); got != nil {
		t.Errorf("TopologySpec.NetworkSegments() = %v, want nil", got)
	}
}

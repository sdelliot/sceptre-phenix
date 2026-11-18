package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/gorilla/mux"

	"phenix/store"
	ifaces "phenix/types/interfaces"
	v1 "phenix/types/version/v1"
	"phenix/web/middleware"
	"phenix/web/rbac"
)

func TestGetTopologyNetworkSegments(t *testing.T) {
	m := store.NewMockStore(gomock.NewController(t))
	m.EXPECT().Get(gomock.Any()).DoAndReturn(func(c *store.Config) error {
		if c.FullName() != "Topology/lab" {
			return store.ErrNotExist
		}

		*c = store.Config{
			Version: "phenix.sandia.gov/v1",
			Kind:    "Topology",
			Metadata: store.ConfigMetadata{
				Name: "lab",
			},
			Spec: map[string]any{
				"nodes": []any{
					map[string]any{
						"general": map[string]any{"hostname": "vm"},
						"network": map[string]any{"interfaces": []any{
							map[string]any{"name": "eth0", "type": "ethernet", "proto": "static", "vlan": "control", "address": "10.0.0.2", "mask": 24, "mac": "00:00:00:00:00:01", "mtu": 1500},
						}},
					},
					map[string]any{
						"external": true,
						"general":  map[string]any{"hostname": "hil"},
						"network": map[string]any{"interfaces": []any{
							map[string]any{"name": "eth0", "bridge": "plant", "vlan": "ics"},
						}},
					},
				},
			},
		}

		return nil
	}).AnyTimes()

	previous := store.DefaultStore
	store.DefaultStore = m //nolint:reassign // monkey patching for test

	t.Cleanup(func() { store.DefaultStore = previous }) //nolint:reassign // monkey patching for test

	role := rbac.Role{Spec: &v1.RoleSpec{Policies: []*v1.PolicySpec{
		{Resources: []string{"topologies"}, ResourceNames: []string{"lab"}, Verbs: []string{"get"}},
	}}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/topologies/lab/network-segments", nil)
	req = mux.SetURLVars(req, map[string]string{"topo": "lab"})
	ctx := context.WithValue(req.Context(), middleware.ContextKeyRole, role)
	ctx = context.WithValue(ctx, middleware.ContextKeyUser, "test-user")

	rec := httptest.NewRecorder()
	GetTopologyNetworkSegments(rec, req.WithContext(ctx))

	if rec.Code != http.StatusOK {
		t.Fatalf("GET network segments status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body)
	}

	var got struct {
		NetworkSegments []ifaces.NetworkSegment `json:"network_segments"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	want := []ifaces.NetworkSegment{
		{
			Bridge: "phenix",
			VLAN:   "control",
			Endpoints: []ifaces.NetworkSegmentEndpoint{
				{Hostname: "vm", Interface: "eth0", Type: "ethernet", Proto: "static", Address: "10.0.0.2", Mask: 24, MAC: "00:00:00:00:00:01", MTU: 1500, External: false},
			},
		},
		{
			Bridge: "plant",
			VLAN:   "ics",
			Endpoints: []ifaces.NetworkSegmentEndpoint{
				{Hostname: "hil", Interface: "eth0", External: true},
			},
		},
	}

	if diff := cmp.Diff(want, got.NetworkSegments); diff != "" {
		t.Errorf("GET network segments mismatch (-want +got):\n%s", diff)
	}
}

func TestGetTopologyNetworkSegmentsRequiresTopologyGet(t *testing.T) {
	role := rbac.Role{Spec: &v1.RoleSpec{}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/topologies/lab/network-segments", nil)
	req = mux.SetURLVars(req, map[string]string{"topo": "lab"})
	ctx := context.WithValue(req.Context(), middleware.ContextKeyRole, role)
	ctx = context.WithValue(ctx, middleware.ContextKeyUser, "test-user")

	rec := httptest.NewRecorder()
	GetTopologyNetworkSegments(rec, req.WithContext(ctx))

	if rec.Code != http.StatusForbidden {
		t.Errorf("GET network segments without permission status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

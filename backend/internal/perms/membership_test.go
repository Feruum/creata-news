package perms

import (
	"context"
	pb "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"google.golang.org/grpc"
	"net"
	"sync"
	"testing"
)

type membershipWriter struct {
	pb.UnimplementedPermissionsServiceServer
	mu     sync.Mutex
	writes []*pb.WriteRelationshipsRequest
}

func (s *membershipWriter) WriteRelationships(_ context.Context, req *pb.WriteRelationshipsRequest) (*pb.WriteRelationshipsResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.writes = append(s.writes, req)
	return &pb.WriteRelationshipsResponse{}, nil
}
func TestSetOrganizationMemberReplacesCapabilitiesAtomically(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	fixture := &membershipWriter{}
	server := grpc.NewServer()
	pb.RegisterPermissionsServiceServer(server, fixture)
	go server.Serve(listener)
	t.Cleanup(server.Stop)
	client, err := NewClient(listener.Addr().String(), "test", true, false, "default")
	if err != nil {
		t.Fatal(err)
	}
	if err := client.SetOrganizationMember(context.Background(), "3", "7", []string{CapabilityViewAnalytics, CapabilityViewAnalytics}); err != nil {
		t.Fatal(err)
	}
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	if len(fixture.writes) != 1 {
		t.Fatalf("replacement used %d writes", len(fixture.writes))
	}
	updates := fixture.writes[0].Updates
	seen := map[string]pb.RelationshipUpdate_Operation{}
	for _, u := range updates {
		r := u.Relationship
		if r.Resource.ObjectId != "3" || r.Subject.Object.ObjectId != "7" {
			t.Fatal("wrong principal")
		}
		if _, ok := seen[r.Relation]; ok {
			t.Fatalf("duplicate update for %s", r.Relation)
		}
		seen[r.Relation] = u.Operation
	}
	if seen["member"] != pb.RelationshipUpdate_OPERATION_TOUCH || seen[CapabilityViewAnalytics] != pb.RelationshipUpdate_OPERATION_TOUCH {
		t.Fatal("missing intended grants")
	}
	for _, cap := range AllCapabilities {
		if cap != CapabilityViewAnalytics && seen[cap] != pb.RelationshipUpdate_OPERATION_DELETE {
			t.Fatalf("stale capability %s not revoked", cap)
		}
	}
}
func TestSetPlatformRoleReplacesAndRevokesAtomically(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	fixture := &membershipWriter{}
	server := grpc.NewServer()
	pb.RegisterPermissionsServiceServer(server, fixture)
	go server.Serve(listener)
	t.Cleanup(server.Stop)
	client, err := NewClient(listener.Addr().String(), "test", true, false, "default")
	if err != nil {
		t.Fatal(err)
	}
	if err := client.SetPlatformRole(context.Background(), "7", "staff"); err != nil {
		t.Fatal(err)
	}
	if err := client.SetPlatformRole(context.Background(), "7", ""); err != nil {
		t.Fatal(err)
	}
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	if len(fixture.writes) != 2 {
		t.Fatalf("platform replacements used %d writes", len(fixture.writes))
	}
	for i, req := range fixture.writes {
		if len(req.Updates) != 2 {
			t.Fatal("role replacement must update both relations")
		}
		for _, update := range req.Updates {
			expected := pb.RelationshipUpdate_OPERATION_DELETE
			if i == 0 && update.Relationship.Relation == "staff" {
				expected = pb.RelationshipUpdate_OPERATION_TOUCH
			}
			if update.Operation != expected {
				t.Fatalf("stale platform grant retained: %v", update)
			}
		}
	}
}

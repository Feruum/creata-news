package perms

import (
	"context"
	pb "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net"
	"testing"
)

type failingLookupGraph struct {
	pb.UnimplementedPermissionsServiceServer
}

func (failingLookupGraph) LookupResources(_ *pb.LookupResourcesRequest, stream pb.PermissionsService_LookupResourcesServer) error {
	if err := stream.Send(&pb.LookupResourcesResponse{ResourceObjectId: "1"}); err != nil {
		return err
	}
	return status.Error(codes.Unavailable, "partial resource lookup")
}
func (failingLookupGraph) LookupSubjects(_ *pb.LookupSubjectsRequest, stream pb.PermissionsService_LookupSubjectsServer) error {
	if err := stream.Send(&pb.LookupSubjectsResponse{Subject: &pb.ResolvedSubject{SubjectObjectId: "1"}}); err != nil {
		return err
	}
	return status.Error(codes.Unavailable, "partial subject lookup")
}
func TestLookupsRejectIncompleteStreams(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer()
	pb.RegisterPermissionsServiceServer(server, failingLookupGraph{})
	go server.Serve(listener)
	t.Cleanup(server.Stop)
	client, err := NewClient(listener.Addr().String(), "test", true, false, "default")
	if err != nil {
		t.Fatal(err)
	}
	for name, lookup := range map[string]func() ([]string, error){"resources": func() ([]string, error) {
		return client.LookupResources(context.Background(), "1", "organization", "view")
	}, "subjects": func() ([]string, error) {
		return client.LookupSubjects(context.Background(), "organization", "1", "manage_settings")
	}} {
		t.Run(name, func(t *testing.T) {
			ids, err := lookup()
			if status.Code(err) != codes.Unavailable || len(ids) != 0 {
				t.Fatalf("incomplete lookup reported success: ids=%v err=%v", ids, err)
			}
		})
	}
}

//go:build integration

package events

import (
	"connectrpc.com/connect"
	"context"
	"fmt"
	pb "github.com/authzed/authzed-go/proto/authzed/api/v1"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/perms"
	"github.com/studyverse/ems-backend/internal/testsupport"
	"google.golang.org/grpc"
	"net"
	"testing"
)

type staleDraftGraph struct {
	pb.UnimplementedPermissionsServiceServer
}

func (staleDraftGraph) CheckPermission(_ context.Context, req *pb.CheckPermissionRequest) (*pb.CheckPermissionResponse, error) {
	r := pb.CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION
	if req.Resource.ObjectType == "platform" {
		r = pb.CheckPermissionResponse_PERMISSIONSHIP_NO_PERMISSION
	}
	return &pb.CheckPermissionResponse{Permissionship: r}, nil
}
func TestAdminEventListAndDraftDetailsRespectCurrentEditors(t *testing.T) {
	db := testsupport.ClonedDB(t)
	ctx := context.Background()
	insert := func(q string, args ...any) int32 {
		t.Helper()
		var id int32
		if err := db.NewRaw(q, args...).Scan(ctx, &id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, q, args...); err != nil {
			t.Fatal(err)
		}
	}
	editor := insert("INSERT INTO users(kratos_id,username,email,password) VALUES('audit-draft-editor','audit-draft-editor','audit-draft-editor@test.local','test') RETURNING id")
	other := insert("INSERT INTO users(kratos_id,username,email,password) VALUES('audit-draft-other','audit-draft-other','audit-draft-other@test.local','test') RETURNING id")
	typ := insert("INSERT INTO organization_types(title) VALUES('Audit draft type') RETURNING id")
	org := insert("INSERT INTO organizations(title,organization_type_id) VALUES('Audit editable draft org',?) RETURNING id", typ)
	outside := insert("INSERT INTO organizations(title,organization_type_id) VALUES('Audit outside draft org',?) RETURNING id", typ)
	role := insert("INSERT INTO role_templates(organization_type_id,key,display_name,is_active) VALUES(?,'draft-editor','Draft editor',true) RETURNING id", typ)
	exec("INSERT INTO role_template_capabilities(role_template_id,capability) VALUES(?,'create_event')", role)
	exec("INSERT INTO organization_members(organization_id,user_id,role_template_id) VALUES(?,?,?)", org, editor, role)
	create := func(title string, creator, orgID int32) int32 {
		return insert("INSERT INTO events(title,description,structured_location,user_id,organization_id,start_time,end_time,status,check_in_secret,check_in_token) VALUES(?,'{}','{}',?,?,NOW()+INTERVAL '1 day',NOW()+INTERVAL '2 days','draft',substr(md5(random()::text),1,16),substr(md5(random()::text),1,16)) RETURNING id", title, creator, orgID)
	}
	hosted := create("Editable hosted draft", other, org)
	collaborating := create("Editable collaborator draft", other, outside)
	own := create("Own draft", editor, outside)
	hidden := create("Unrelated draft", other, outside)
	exec("INSERT INTO event_collaborators(event_id,organization_id) VALUES(?,?)", collaborating, org)
	for _, id := range []int32{hosted, collaborating, own, hidden} {
		exec("INSERT INTO event_slugs(event_id,slug,is_current) VALUES(?,?,true)", id, fmt.Sprintf("audit-draft-%d", id))
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer()
	pb.RegisterPermissionsServiceServer(server, staleDraftGraph{})
	go server.Serve(listener)
	t.Cleanup(server.Stop)
	client, err := perms.NewClient(listener.Addr().String(), "test", true, false, "default")
	if err != nil {
		t.Fatal(err)
	}
	s := NewService(db, client, authz.NewEnforcer(db, client), nil, nil, "", "")
	t.Run("anonymous list", func(t *testing.T) {
		_, err := s.ListEventsForAdmin(ctx, connect.NewRequest(&eventsv1.ListEventsForAdminRequest{}))
		if connect.CodeOf(err) != connect.CodeUnauthenticated {
			t.Fatalf("anonymous draft list returned %v", err)
		}
	})
	ctx = context.WithValue(ctx, auth.UserIDKey, "audit-draft-editor")
	t.Run("authorized list and total", func(t *testing.T) {
		res, err := s.ListEventsForAdmin(ctx, connect.NewRequest(&eventsv1.ListEventsForAdminRequest{Limit: 1}))
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Msg.Events) != 1 || res.Msg.Total != 3 {
			t.Fatalf("list count leaked or hid drafts: len=%d total=%d", len(res.Msg.Events), res.Msg.Total)
		}
	})
	for _, id := range []int32{hosted, collaborating} {
		t.Run(fmt.Sprintf("editable detail %d", id), func(t *testing.T) {
			res, err := s.GetEvent(ctx, connect.NewRequest(&eventsv1.GetEventRequest{Id: id}))
			if err != nil {
				t.Fatal(err)
			}
			if !res.Msg.CanEdit {
				t.Fatal("editor cannot edit")
			}
			if _, err := s.GetEventBySlug(ctx, connect.NewRequest(&eventsv1.GetEventBySlugRequest{Slug: fmt.Sprintf("audit-draft-%d", id)})); err != nil {
				t.Fatal(err)
			}
		})
	}
	if _, err := s.GetEvent(ctx, connect.NewRequest(&eventsv1.GetEventRequest{Id: hidden})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("unrelated draft disclosed: %v", err)
	}
	exec("UPDATE organization_members SET valid_until=NOW()-INTERVAL '1 hour' WHERE user_id=?", editor)
	for _, id := range []int32{hosted, collaborating} {
		if _, err := s.GetEvent(ctx, connect.NewRequest(&eventsv1.GetEventRequest{Id: id})); connect.CodeOf(err) != connect.CodeNotFound {
			t.Fatalf("expired editor draft disclosed: %v", err)
		}
	}
	res, err := s.ListEventsForAdmin(ctx, connect.NewRequest(&eventsv1.ListEventsForAdminRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if res.Msg.Total != 1 || len(res.Msg.Events) != 1 || res.Msg.Events[0].Id != own {
		t.Fatalf("expired grants listed: %+v", res.Msg)
	}
	exec("DELETE FROM organization_members WHERE user_id=?", editor)
	if _, err := s.GetEventBySlug(ctx, connect.NewRequest(&eventsv1.GetEventBySlugRequest{Slug: fmt.Sprintf("audit-draft-%d", hosted)})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("revoked editor draft disclosed: %v", err)
	}
}

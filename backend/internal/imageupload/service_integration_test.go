//go:build integration

package imageupload

import (
	"bytes"
	"connectrpc.com/connect"
	"context"
	"encoding/base64"
	"fmt"
	pb "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/perms"
	"github.com/studyverse/ems-backend/internal/storage"
	"github.com/studyverse/ems-backend/internal/testsupport"
	"google.golang.org/grpc"
	"image/png"
	"net"
	"strings"
	"testing"
	"time"
)

type uploadGraph struct {
	pb.UnimplementedPermissionsServiceServer
	staff string
}

func (g uploadGraph) CheckPermission(_ context.Context, r *pb.CheckPermissionRequest) (*pb.CheckPermissionResponse, error) {
	grant := r.Resource.ObjectType != "platform" || r.Subject.Object.ObjectId == g.staff
	result := pb.CheckPermissionResponse_PERMISSIONSHIP_NO_PERMISSION
	if grant {
		result = pb.CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION
	}
	return &pb.CheckPermissionResponse{Permissionship: result}, nil
}

type recordingStorage struct {
	keys []string
	data [][]byte
	fail bool
}

func (s *recordingStorage) UploadObject(_ context.Context, key, mime string, data []byte) error {
	if s.fail {
		return fmt.Errorf("storage down")
	}
	if mime != "image/png" {
		return fmt.Errorf("unsafe mime")
	}
	s.keys = append(s.keys, key)
	s.data = append(s.data, bytes.Clone(data))
	return nil
}

func (s *recordingStorage) GeneratePresignedUploadURL(context.Context, string, string, time.Duration) (*storage.UploadURLResult, error) {
	panic("presign must never be called")
}
func (s *recordingStorage) DeleteObject(context.Context, string) error {
	panic("upload must never delete another image")
}
func (s *recordingStorage) GetPublicURL(key string) string { return "https://images.test/" + key }
func TestImageUploadsRequireCurrentEntityPermissionsAndOnlyStoreValidatedBytes(t *testing.T) {
	db := testsupport.ClonedDB(t)
	base := context.Background()
	insert := func(q string, args ...any) int32 {
		t.Helper()
		var id int32
		if err := db.NewRaw(q, args...).Scan(base, &id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(base, q, args...); err != nil {
			t.Fatal(err)
		}
	}
	editor := insert("INSERT INTO users(kratos_id,username,email,password) VALUES('upload-editor','upload-editor','upload-editor@test.invalid','test') RETURNING id")
	outsider := insert("INSERT INTO users(kratos_id,username,email,password) VALUES('upload-outsider','upload-outsider','upload-outsider@test.invalid','test') RETURNING id")
	viewer := insert("INSERT INTO users(kratos_id,username,email,password) VALUES('upload-viewer','upload-viewer','upload-viewer@test.invalid','test') RETURNING id")
	_ = viewer
	staff := insert("INSERT INTO users(kratos_id,username,email,password) VALUES('upload-staff','upload-staff','upload-staff@test.invalid','test') RETURNING id")
	kind := insert("INSERT INTO organization_types(title) VALUES('Upload test') RETURNING id")
	org := insert("INSERT INTO organizations(title,organization_type_id) VALUES('Upload test',?) RETURNING id", kind)
	role := insert("INSERT INTO role_templates(organization_type_id,key,display_name,is_active) VALUES(?,'upload-editor','Upload editor',true) RETURNING id", kind)
	exec("INSERT INTO role_template_capabilities(role_template_id,capability) VALUES(?,'create_event'),(?,'manage_settings')", role, role)
	exec("INSERT INTO organization_members(organization_id,user_id,role_template_id) VALUES(?,?,?)", org, editor, role)
	exec("INSERT INTO platform_user_roles(user_id,role) VALUES(?,'staff')", staff)
	event := insert("INSERT INTO events(title,description,structured_location,user_id,organization_id,start_time,end_time,status,check_in_secret,check_in_token) VALUES('Upload event','{}','{}',?,?,NOW()+INTERVAL '1 day',NOW()+INTERVAL '2 days','draft','testsecret','uploadtoken') RETURNING id", outsider, org)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	graph := grpc.NewServer()
	pb.RegisterPermissionsServiceServer(graph, uploadGraph{staff: fmt.Sprint(staff)})
	go func() { _ = graph.Serve(listener) }()
	t.Cleanup(graph.Stop)
	client, err := perms.NewClient(listener.Addr().String(), "test", true, false, "upload-test")
	if err != nil {
		t.Fatal(err)
	}
	provider := &recordingStorage{}
	svc := Service{DB: db, Authz: authz.NewEnforcer(db, client), Storage: provider}
	pngData, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatal(err)
	}
	editorCtx := context.WithValue(base, auth.UserIDKey, "upload-editor")
	otherCtx := context.WithValue(base, auth.UserIDKey, "upload-outsider")
	viewerCtx := context.WithValue(base, auth.UserIDKey, "upload-viewer")
	staffCtx := context.WithValue(base, auth.UserIDKey, "upload-staff")
	check := func(t *testing.T, url, key string, err error, prefix string) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(key, prefix) || !strings.HasSuffix(key, ".png") || url != "https://images.test/"+key {
			t.Fatalf("unsafe object scope: %s %s", key, url)
		}
		if _, err := png.Decode(bytes.NewReader(provider.data[len(provider.data)-1])); err != nil {
			t.Fatal(err)
		}
	}
	t.Run("new draft requires organization create capability", func(t *testing.T) {
		url, key, err := svc.UploadEvent(editorCtx, pngData, "image/png", 0, org)
		check(t, url, key, err, fmt.Sprintf("events/org-%d/u%d/", org, editor))
	})
	t.Run("existing event requires edit", func(t *testing.T) {
		url, key, err := svc.UploadEvent(editorCtx, pngData, "image/png", event, 0)
		check(t, url, key, err, fmt.Sprintf("events/%d/u%d/", event, editor))
	})
	t.Run("numeric creator may upload without org membership", func(t *testing.T) {
		url, key, err := svc.UploadEvent(otherCtx, pngData, "image/png", event, 0)
		check(t, url, key, err, fmt.Sprintf("events/%d/u%d/", event, outsider))
	})
	t.Run("existing organization requires settings capability", func(t *testing.T) {
		url, key, err := svc.UploadOrganization(editorCtx, pngData, "image/png", org)
		check(t, url, key, err, fmt.Sprintf("organizations/%d/u%d/", org, editor))
	})
	t.Run("precreation organization is staff only", func(t *testing.T) {
		before := len(provider.keys)
		_, _, err := svc.UploadOrganization(editorCtx, pngData, "image/png", 0)
		if connect.CodeOf(err) != connect.CodePermissionDenied || len(provider.keys) != before {
			t.Fatalf("nonstaff precreation allowed: %v", err)
		}
		url, key, err := svc.UploadOrganization(staffCtx, pngData, "image/png", 0)
		check(t, url, key, err, fmt.Sprintf("organizations/drafts/u%d/", staff))
	})
	for _, c := range []struct {
		name       string
		ctx        context.Context
		event, org int32
		mime       string
		data       []byte
		want       connect.Code
	}{
		{"unauthenticated", base, 0, org, "image/png", pngData, connect.CodeUnauthenticated},
		{"missing target", editorCtx, 0, 0, "image/png", pngData, connect.CodeInvalidArgument},
		{"ambiguous target", editorCtx, event, org, "image/png", pngData, connect.CodeInvalidArgument},
		{"foreign organization", otherCtx, 0, org, "image/png", pngData, connect.CodePermissionDenied},
		{"foreign event", viewerCtx, event, 0, "image/png", pngData, connect.CodePermissionDenied},
		{"missing event even for staff", staffCtx, 2147483647, 0, "image/png", pngData, connect.CodeNotFound},
		{"missing org even for staff", staffCtx, 0, 2147483647, "image/png", pngData, connect.CodeNotFound},
		{"invalid content", editorCtx, 0, org, "image/png", []byte("bad image"), connect.CodeInvalidArgument},
		{"oversize bytes", editorCtx, 0, org, "image/png", make([]byte, storage.MaxImageBytes+1), connect.CodeInvalidArgument},
	} {
		t.Run(c.name, func(t *testing.T) {
			before := len(provider.keys)
			_, _, err := svc.UploadEvent(c.ctx, c.data, c.mime, c.event, c.org)
			if connect.CodeOf(err) != c.want || len(provider.keys) != before {
				t.Fatalf("reject without writes: got %v want %v", err, c.want)
			}
		})
	}
	t.Run("stale graph cannot override expired SQL capability", func(t *testing.T) {
		exec("UPDATE organization_members SET valid_until=NOW()-INTERVAL '1 day' WHERE user_id=?", editor)
		before := len(provider.keys)
		_, _, err := svc.UploadEvent(editorCtx, pngData, "image/png", event, 0)
		if connect.CodeOf(err) != connect.CodePermissionDenied || len(provider.keys) != before {
			t.Fatalf("expired grant allowed: %v", err)
		}
		_, _, err = svc.UploadOrganization(editorCtx, pngData, "image/png", org)
		if connect.CodeOf(err) != connect.CodePermissionDenied || len(provider.keys) != before {
			t.Fatalf("expired settings grant allowed: %v", err)
		}
	})
	t.Run("provider failure never returns a public URL", func(t *testing.T) {
		provider.fail = true
		url, key, err := svc.UploadEvent(otherCtx, pngData, "image/png", event, 0)
		if connect.CodeOf(err) != connect.CodeInternal || url != "" || key != "" {
			t.Fatalf("failed storage returned URL: %s %s %v", url, key, err)
		}
	})
}

package main

import (
	"context"
	"log/slog"
	"os"

	pb "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"github.com/authzed/authzed-go/v1"
	"github.com/authzed/grpcutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))

	endpoint := envOrFlag("SPICEDB_ENDPOINT", "endpoint", "localhost:50051")
	token := envOrFlag("SPICEDB_TOKEN", "token", "")
	platformID := envOrFlag("PLATFORM_ID", "platform-id", "default")
	insecureFlag := os.Getenv("SPICEDB_INSECURE") == "true"

	if token == "" {
		slog.Error("SPICEDB_TOKEN or --token is required")
		os.Exit(1)
	}

	ctx := context.Background()

	var opts []grpc.DialOption
	if insecureFlag {
		opts = append(opts,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpcutil.WithInsecureBearerToken(token),
		)
	} else {
		systemCerts, err := grpcutil.WithSystemCerts(grpcutil.VerifyCA)
		if err != nil {
			slog.Error("failed to get system certs", "error", err)
			os.Exit(1)
		}
		opts = append(opts,
			systemCerts,
			grpcutil.WithBearerToken(token),
		)
	}

	client, err := authzed.NewClient(endpoint, opts...)
	if err != nil {
		slog.Error("failed to create SpiceDB client", "error", err)
		os.Exit(1)
	}

	// Step 1: Read all platform#admin@user:* tuples
	slog.Info("reading admin tuples", "platform_id", platformID)

	stream, err := client.ReadRelationships(ctx, &pb.ReadRelationshipsRequest{
		RelationshipFilter: &pb.RelationshipFilter{
			ResourceType:       "platform",
			OptionalResourceId: platformID,
			OptionalRelation:   "admin",
			OptionalSubjectFilter: &pb.SubjectFilter{
				SubjectType: "user",
			},
		},
	})
	if err != nil {
		slog.Error("failed to read admin relationships", "error", err)
		os.Exit(1)
	}

	var adminSubjects []string
	for {
		resp, err := stream.Recv()
		if err != nil {
			break
		}
		subjectID := resp.Relationship.Subject.Object.ObjectId
		adminSubjects = append(adminSubjects, subjectID)
		slog.Info("found admin tuple", "user_id", subjectID)
	}

	slog.Info("admin tuples found", "count", len(adminSubjects))

	if len(adminSubjects) == 0 {
		slog.Info("no admin tuples to migrate, done")
		os.Exit(0)
	}

	// Step 2: TOUCH-write each as platform#administrator@user:*
	var touchUpdates []*pb.RelationshipUpdate
	for _, subjectID := range adminSubjects {
		touchUpdates = append(touchUpdates, &pb.RelationshipUpdate{
			Operation: pb.RelationshipUpdate_OPERATION_TOUCH,
			Relationship: &pb.Relationship{
				Resource: &pb.ObjectReference{
					ObjectType: "platform",
					ObjectId:   platformID,
				},
				Relation: "administrator",
				Subject: &pb.SubjectReference{
					Object: &pb.ObjectReference{
						ObjectType: "user",
						ObjectId:   subjectID,
					},
				},
			},
		})
	}

	slog.Info("writing administrator tuples (TOUCH)", "count", len(touchUpdates))
	_, err = client.WriteRelationships(ctx, &pb.WriteRelationshipsRequest{
		Updates: touchUpdates,
	})
	if err != nil {
		slog.Error("failed to write administrator relationships", "error", err)
		os.Exit(1)
	}
	slog.Info("administrator tuples written successfully")

	// Step 3: Delete the old admin tuples
	var deleteUpdates []*pb.RelationshipUpdate
	for _, subjectID := range adminSubjects {
		deleteUpdates = append(deleteUpdates, &pb.RelationshipUpdate{
			Operation: pb.RelationshipUpdate_OPERATION_DELETE,
			Relationship: &pb.Relationship{
				Resource: &pb.ObjectReference{
					ObjectType: "platform",
					ObjectId:   platformID,
				},
				Relation: "admin",
				Subject: &pb.SubjectReference{
					Object: &pb.ObjectReference{
						ObjectType: "user",
						ObjectId:   subjectID,
					},
				},
			},
		})
	}

	slog.Info("deleting admin tuples", "count", len(deleteUpdates))
	_, err = client.WriteRelationships(ctx, &pb.WriteRelationshipsRequest{
		Updates: deleteUpdates,
	})
	if err != nil {
		slog.Error("failed to delete admin relationships", "error", err)
		os.Exit(1)
	}
	slog.Info("admin tuples deleted successfully")

	slog.Info("backfill complete", "migrated", len(adminSubjects), "platform_id", platformID)
}

func envOrFlag(envKey, _ string, defaultValue string) string {
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	return defaultValue
}

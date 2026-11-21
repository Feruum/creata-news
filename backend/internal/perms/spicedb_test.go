package perms

import "testing"

func TestBuildPlatformRoleRelationships_AdministratorWritesSingleRelation(t *testing.T) {
	rels := buildPlatformRoleRelationships("platform-1", "42", "administrator")

	if len(rels) != 1 {
		t.Fatalf("expected 1 relationship, got %d", len(rels))
	}

	if rels[0].Relation != "administrator" {
		t.Fatalf("relation = %q, want %q", rels[0].Relation, "administrator")
	}

	if rels[0].Resource != "platform" || rels[0].ResourceID != "platform-1" || rels[0].SubjectType != "user" || rels[0].SubjectID != "42" {
		t.Fatalf("unexpected relationship payload: %+v", rels[0])
	}
}

func TestBuildPlatformRoleRelationships_StaffWritesOnlyStaff(t *testing.T) {
	rels := buildPlatformRoleRelationships("platform-1", "42", "staff")

	if len(rels) != 1 {
		t.Fatalf("expected 1 relationship, got %d", len(rels))
	}
	if rels[0].Relation != "staff" {
		t.Fatalf("relation = %q, want %q", rels[0].Relation, "staff")
	}
}

func TestBuildPlatformRoleDeletionRelationships_RemovesCurrentRoles(t *testing.T) {
	rels := buildPlatformRoleDeletionRelationships("platform-1", "42")

	if len(rels) != 2 {
		t.Fatalf("expected 2 relationships, got %d", len(rels))
	}

	want := []string{"administrator", "staff"}
	for i, relation := range want {
		if rels[i].Relation != relation {
			t.Fatalf("relation[%d] = %q, want %q", i, rels[i].Relation, relation)
		}
	}
}

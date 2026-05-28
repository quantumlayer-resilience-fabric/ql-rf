package integration

import (
	"testing"

	"github.com/google/uuid"
)

// This file provides DB seeding helpers shared by the integration tests. They
// insert the parent rows that foreign keys require (organizations, images,
// assets, inspec profiles) so tests can exercise real persistence instead of
// inserting orphan rows with random UUIDs. Each helper registers cleanup; org
// deletion cascades to its child rows, so most cleanup is implicit.

// seedOrg inserts a throwaway organization and returns its ID.
func seedOrg(t *testing.T) uuid.UUID {
	t.Helper()
	id := uuid.New()
	suffix := id.String()[:8]
	if _, err := testDB.Exec(
		"INSERT INTO organizations (id, name, slug) VALUES ($1, $2, $3)",
		id, "it-org-"+suffix, "it-org-"+suffix,
	); err != nil {
		t.Fatalf("seed organization: %v", err)
	}
	t.Cleanup(func() {
		if _, err := testDB.Exec("DELETE FROM organizations WHERE id = $1", id); err != nil {
			t.Logf("cleanup organization %s: %v", id, err)
		}
	})
	return id
}

// seedImage inserts an image owned by orgID and returns its ID. Cleanup happens
// via the owning org's ON DELETE CASCADE.
func seedImage(t *testing.T, orgID uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	if _, err := testDB.Exec(
		"INSERT INTO images (id, org_id, family, version) VALUES ($1, $2, $3, '1.0.0')",
		id, orgID, "it-fam-"+id.String()[:8],
	); err != nil {
		t.Fatalf("seed image: %v", err)
	}
	return id
}

// seedAsset inserts an asset owned by orgID and returns its ID. Cleanup happens
// via the owning org's ON DELETE CASCADE.
func seedAsset(t *testing.T, orgID uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	if _, err := testDB.Exec(
		"INSERT INTO assets (id, org_id, platform, instance_id) VALUES ($1, $2, 'aws', $3)",
		id, orgID, "it-asset-"+id.String()[:8],
	); err != nil {
		t.Fatalf("seed asset: %v", err)
	}
	return id
}

// seedInspecProfile inserts an InSpec profile bound to an existing (pre-seeded)
// compliance framework and returns its ID.
func seedInspecProfile(t *testing.T) uuid.UUID {
	t.Helper()
	var frameworkID uuid.UUID
	if err := testDB.QueryRow("SELECT id FROM compliance_frameworks LIMIT 1").Scan(&frameworkID); err != nil {
		t.Fatalf("look up compliance framework (migrations seed these): %v", err)
	}
	id := uuid.New()
	if _, err := testDB.Exec(
		"INSERT INTO inspec_profiles (id, name, version, title, framework_id) VALUES ($1, $2, '1.0.0', 'Test Profile', $3)",
		id, "it-profile-"+id.String()[:8], frameworkID,
	); err != nil {
		t.Fatalf("seed inspec profile: %v", err)
	}
	t.Cleanup(func() {
		if _, err := testDB.Exec("DELETE FROM inspec_profiles WHERE id = $1", id); err != nil {
			t.Logf("cleanup inspec profile %s: %v", id, err)
		}
	})
	return id
}

// existingComplianceControlID returns the ID of a pre-seeded compliance control
// (migrations populate the CIS/SOC2/etc. control catalog).
func existingComplianceControlID(t *testing.T) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := testDB.QueryRow("SELECT id FROM compliance_controls LIMIT 1").Scan(&id); err != nil {
		t.Fatalf("look up compliance control (migrations seed these): %v", err)
	}
	return id
}

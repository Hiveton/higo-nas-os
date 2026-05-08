package settings

import "testing"

func TestStoreRestoresDefaults(t *testing.T) {
	store := NewStore()

	updated, err := store.Update(Settings{
		Model: ModelPolicy{Mode: ModelModeFamilyHybrid, CloudEnabled: true},
		Privacy: PrivacyPolicy{
			SensitiveDataLocalOnly: true,
			AuditRetentionDays:     30,
		},
	})
	if err != nil {
		t.Fatalf("update settings: %v", err)
	}
	if !updated.Model.CloudEnabled {
		t.Fatal("expected cloud to be enabled before restore")
	}

	restored, err := store.RestoreDefaults()
	if err != nil {
		t.Fatalf("restore settings: %v", err)
	}
	if restored.Model.Mode != ModelModeFamilyHybrid || !restored.Privacy.SensitiveDataLocalOnly {
		t.Fatalf("unexpected restored defaults: %#v", restored)
	}
}

func TestStoreDisablesCloudInEnterpriseLocalMode(t *testing.T) {
	store := NewStore()

	updated, err := store.Update(Settings{
		Model: ModelPolicy{Mode: ModelModeEnterpriseLocal, CloudEnabled: true},
		Privacy: PrivacyPolicy{
			SensitiveDataLocalOnly: true,
			AuditRetentionDays:     90,
		},
	})
	if err != nil {
		t.Fatalf("update settings: %v", err)
	}
	if updated.Model.CloudEnabled {
		t.Fatalf("expected enterprise local mode to disable cloud, got %#v", updated.Model)
	}
}

func TestStoreRequiresSensitiveDataLocalOnly(t *testing.T) {
	store := NewStore()

	_, err := store.Update(Settings{
		Model: ModelPolicy{Mode: ModelModeFamilyHybrid, CloudEnabled: true},
		Privacy: PrivacyPolicy{
			SensitiveDataLocalOnly: false,
			AuditRetentionDays:     90,
		},
	})
	if err == nil {
		t.Fatal("expected sensitive data local-only validation error")
	}
}

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
	if restored.UI.Theme != "auto" || restored.UI.DockPosition != "bottom" || restored.UI.WindowRadius != "default" {
		t.Fatalf("unexpected restored ui defaults: %#v", restored.UI)
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

func TestStorePersistsUISettings(t *testing.T) {
	store := NewStore()

	updated, err := store.Update(Settings{
		Model: ModelPolicy{Mode: ModelModeFamilyHybrid, CloudEnabled: true},
		Privacy: PrivacyPolicy{
			SensitiveDataLocalOnly: true,
			AuditRetentionDays:     90,
		},
		UI: UIPolicy{
			Theme:        "dark",
			WindowRadius: "rounded",
			DockPosition: "left",
			DockStyle:    "side",
			DockIconSize: "small",
		},
	})
	if err != nil {
		t.Fatalf("update settings: %v", err)
	}
	if updated.UI.Theme != "dark" || updated.UI.DockPosition != "left" || updated.UI.DockStyle != "side" {
		t.Fatalf("unexpected ui settings: %#v", updated.UI)
	}
}

func TestStoreRejectsInvalidUISettings(t *testing.T) {
	store := NewStore()

	_, err := store.Update(Settings{
		Model: ModelPolicy{Mode: ModelModeFamilyHybrid},
		Privacy: PrivacyPolicy{
			SensitiveDataLocalOnly: true,
			AuditRetentionDays:     90,
		},
		UI: UIPolicy{
			Theme:        "light",
			WindowRadius: "default",
			DockPosition: "diagonal",
			DockStyle:    "floating",
			DockIconSize: "default",
		},
	})
	if err == nil {
		t.Fatal("expected invalid ui validation error")
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

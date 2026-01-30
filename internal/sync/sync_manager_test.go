package sync

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// TestSyncManagerService_NewMethods tests the new methods added for sync configuration management
func TestSyncManagerService_NewMethods(t *testing.T) {
	// Create a mock repository
	repo := &mockRepository{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce log noise in tests

	// Create sync manager service
	syncManager := NewSyncManager(repo, logger, nil)

	ctx := context.Background()

	// Test GetRemoteTables
	t.Run("GetRemoteTables", func(t *testing.T) {
		_, err := syncManager.GetRemoteTables(ctx, "test-connection-id", "test_db")
		if err == nil {
			t.Error("Expected error from mock repository")
		}
	})

	// Test GetRemoteTableSchema
	t.Run("GetRemoteTableSchema", func(t *testing.T) {
		_, err := syncManager.GetRemoteTableSchema(ctx, "test-connection-id", "test_db", "test-table")
		if err == nil {
			t.Error("Expected error from mock repository")
		}
	})

	// Test AddTableMapping
	t.Run("AddTableMapping", func(t *testing.T) {
		mapping := &TableMapping{
			SourceTable: "source_table",
			TargetTable: "target_table",
			SyncMode:    SyncModeFull,
			Enabled:     true,
		}

		err := syncManager.AddTableMapping(ctx, "sync-config-id", mapping)
		if err == nil {
			t.Error("Expected error from mock repository")
		}
	})

	// Test UpdateTableMapping
	t.Run("UpdateTableMapping", func(t *testing.T) {
		mapping := &TableMapping{
			SourceTable: "source_table",
			TargetTable: "target_table",
			SyncMode:    SyncModeIncremental,
			Enabled:     false,
		}

		err := syncManager.UpdateTableMapping(ctx, "mapping-id", mapping)
		if err == nil {
			t.Error("Expected error from mock repository")
		}
	})

	// Test RemoveTableMapping
	t.Run("RemoveTableMapping", func(t *testing.T) {
		err := syncManager.RemoveTableMapping(ctx, "mapping-id")
		if err == nil {
			t.Error("Expected error from mock repository")
		}
	})

	// Test GetTableMappings
	t.Run("GetTableMappings", func(t *testing.T) {
		_, err := syncManager.GetTableMappings(ctx, "sync-config-id")
		if err == nil {
			t.Error("Expected error from mock repository")
		}
	})

	// Test ToggleTableMapping
	t.Run("ToggleTableMapping", func(t *testing.T) {
		err := syncManager.ToggleTableMapping(ctx, "mapping-id", true)
		if err == nil {
			t.Error("Expected error from mock repository")
		}
	})

	// Test SetTableSyncMode
	t.Run("SetTableSyncMode", func(t *testing.T) {
		err := syncManager.SetTableSyncMode(ctx, "mapping-id", SyncModeIncremental)
		if err == nil {
			t.Error("Expected error from mock repository")
		}
	})
}

// TestSyncManagerService_ValidateTableMapping tests table mapping validation
func TestSyncManagerService_ValidateTableMapping(t *testing.T) {
	repo := &mockRepository{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	service := &SyncManagerService{
		Service: NewService(repo, logger, nil),
	}

	tests := []struct {
		name    string
		mapping *TableMapping
		wantErr bool
	}{
		{
			name: "valid mapping",
			mapping: &TableMapping{
				SourceTable: "source_table",
				TargetTable: "target_table",
				SyncMode:    SyncModeFull,
				Enabled:     true,
			},
			wantErr: false,
		},
		{
			name: "empty source table",
			mapping: &TableMapping{
				SourceTable: "",
				TargetTable: "target_table",
				SyncMode:    SyncModeFull,
				Enabled:     true,
			},
			wantErr: true,
		},
		{
			name: "empty target table",
			mapping: &TableMapping{
				SourceTable: "source_table",
				TargetTable: "",
				SyncMode:    SyncModeFull,
				Enabled:     true,
			},
			wantErr: true,
		},
		{
			name: "invalid sync mode",
			mapping: &TableMapping{
				SourceTable: "source_table",
				TargetTable: "target_table",
				SyncMode:    "invalid_mode",
				Enabled:     true,
			},
			wantErr: true,
		},
		{
			name: "invalid target table name",
			mapping: &TableMapping{
				SourceTable: "source_table",
				TargetTable: "123invalid",
				SyncMode:    SyncModeFull,
				Enabled:     true,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateTableMapping(tt.mapping)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateTableMapping() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestSyncManagerService_EnhancedValidation tests the enhanced sync config validation
func TestSyncManagerService_EnhancedValidation(t *testing.T) {
	repo := &mockRepository{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	service := &SyncManagerService{
		Service: NewService(repo, logger, nil),
	}

	tests := []struct {
		name    string
		config  *SyncConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &SyncConfig{
				ID:                 uuid.New().String(),
				SourceConnectionID: uuid.New().String(),
				TargetConnectionID: uuid.New().String(),
				SourceDatabase:     "source_db",
				TargetDatabase:     "target_db",
				Name:               "test-sync",
				SyncMode:           SyncModeFull,
				Enabled:            true,
				Tables: []*TableMapping{
					{
						SourceTable: "source_table",
						TargetTable: "target_table",
						SyncMode:    SyncModeFull,
						Enabled:     true,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid sync mode",
			config: &SyncConfig{
				ID:                 uuid.New().String(),
				SourceConnectionID: uuid.New().String(),
				TargetConnectionID: uuid.New().String(),
				SourceDatabase:     "source_db",
				TargetDatabase:     "target_db",
				Name:               "test-sync",
				SyncMode:           "invalid_mode",
				Enabled:            true,
				Tables: []*TableMapping{
					{
						SourceTable: "source_table",
						TargetTable: "target_table",
						SyncMode:    SyncModeFull,
						Enabled:     true,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid table mapping sync mode",
			config: &SyncConfig{
				ID:                 uuid.New().String(),
				SourceConnectionID: uuid.New().String(),
				TargetConnectionID: uuid.New().String(),
				SourceDatabase:     "source_db",
				TargetDatabase:     "target_db",
				Name:               "test-sync",
				SyncMode:           SyncModeFull,
				Enabled:            true,
				Tables: []*TableMapping{
					{
						SourceTable: "source_table",
						TargetTable: "target_table",
						SyncMode:    "invalid_mode",
						Enabled:     true,
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateSyncConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSyncConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestSyncManagerService_UpdateTableMappings tests the table mapping update logic
func TestSyncManagerService_UpdateTableMappings(t *testing.T) {
	repo := &mockRepository{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	service := &SyncManagerService{
		Service: NewService(repo, logger, nil),
	}

	ctx := context.Background()
	syncConfigID := uuid.New().String()

	// Test with empty mappings (should not error with mock)
	existingMappings := []*TableMapping{}
	newMappings := []*TableMapping{
		{
			SourceTable: "new_table",
			TargetTable: "new_target",
			SyncMode:    SyncModeFull,
			Enabled:     true,
		},
	}

	err := service.updateTableMappings(ctx, syncConfigID, existingMappings, newMappings)
	// With mock repository, this should return an error
	if err == nil {
		t.Error("Expected error from mock repository")
	}
}

// TestSyncManagerService_SetTableSyncModeValidation tests sync mode validation
func TestSyncManagerService_SetTableSyncModeValidation(t *testing.T) {
	repo := &mockRepository{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	service := &SyncManagerService{
		Service: NewService(repo, logger, nil),
	}

	ctx := context.Background()

	// Test invalid sync mode
	err := service.SetTableSyncMode(ctx, "mapping-id", "invalid_mode")
	if err == nil {
		t.Error("Expected error for invalid sync mode")
	}

	// Test valid sync modes
	validModes := []SyncMode{SyncModeFull, SyncModeIncremental}
	for _, mode := range validModes {
		err := service.SetTableSyncMode(ctx, "mapping-id", mode)
		// Should get error from mock repository, not from validation
		if err == nil {
			t.Error("Expected error from mock repository")
		}
	}
}

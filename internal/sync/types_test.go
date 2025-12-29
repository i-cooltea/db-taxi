package sync

import (
	"encoding/json"
	"testing"
	"time"
)

// TestSyncModeConstants tests that SyncMode constants are properly defined
func TestSyncModeConstants(t *testing.T) {
	tests := []struct {
		mode     SyncMode
		expected string
	}{
		{SyncModeFull, "full"},
		{SyncModeIncremental, "incremental"},
	}

	for _, test := range tests {
		if string(test.mode) != test.expected {
			t.Errorf("Expected SyncMode %s to equal %s, got %s",
				test.expected, test.expected, string(test.mode))
		}
	}
}

// TestConflictResolutionConstants tests that ConflictResolution constants are properly defined
func TestConflictResolutionConstants(t *testing.T) {
	tests := []struct {
		resolution ConflictResolution
		expected   string
	}{
		{ConflictResolutionSkip, "skip"},
		{ConflictResolutionOverwrite, "overwrite"},
		{ConflictResolutionError, "error"},
	}

	for _, test := range tests {
		if string(test.resolution) != test.expected {
			t.Errorf("Expected ConflictResolution %s to equal %s, got %s",
				test.expected, test.expected, string(test.resolution))
		}
	}
}

// TestJobStatusConstants tests that JobStatus constants are properly defined
func TestJobStatusConstants(t *testing.T) {
	tests := []struct {
		status   JobStatus
		expected string
	}{
		{JobStatusPending, "pending"},
		{JobStatusRunning, "running"},
		{JobStatusCompleted, "completed"},
		{JobStatusFailed, "failed"},
		{JobStatusCancelled, "cancelled"},
	}

	for _, test := range tests {
		if string(test.status) != test.expected {
			t.Errorf("Expected JobStatus %s to equal %s, got %s",
				test.expected, test.expected, string(test.status))
		}
	}
}

// TestConnectionConfigStructure tests ConnectionConfig struct fields and JSON serialization
func TestConnectionConfigStructure(t *testing.T) {
	now := time.Now()
	config := &ConnectionConfig{
		ID:          "test-id",
		Name:        "test-connection",
		Host:        "localhost",
		Port:        3306,
		Username:    "testuser",
		Password:    "testpass",
		Database:    "testdb",
		LocalDBName: "local_testdb",
		SSL:         true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Test field access
	if config.ID != "test-id" {
		t.Errorf("Expected ID 'test-id', got %s", config.ID)
	}
	if config.Name != "test-connection" {
		t.Errorf("Expected Name 'test-connection', got %s", config.Name)
	}
	if config.Host != "localhost" {
		t.Errorf("Expected Host 'localhost', got %s", config.Host)
	}
	if config.Port != 3306 {
		t.Errorf("Expected Port 3306, got %d", config.Port)
	}
	if config.Username != "testuser" {
		t.Errorf("Expected Username 'testuser', got %s", config.Username)
	}
	if config.Password != "testpass" {
		t.Errorf("Expected Password 'testpass', got %s", config.Password)
	}
	if config.Database != "testdb" {
		t.Errorf("Expected Database 'testdb', got %s", config.Database)
	}
	if config.LocalDBName != "local_testdb" {
		t.Errorf("Expected LocalDBName 'local_testdb', got %s", config.LocalDBName)
	}
	if !config.SSL {
		t.Error("Expected SSL to be true")
	}

	// Test JSON serialization
	jsonData, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal ConnectionConfig to JSON: %v", err)
	}

	var unmarshaled ConnectionConfig
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal ConnectionConfig from JSON: %v", err)
	}

	if unmarshaled.ID != config.ID {
		t.Errorf("JSON roundtrip failed for ID: expected %s, got %s", config.ID, unmarshaled.ID)
	}
	if unmarshaled.Name != config.Name {
		t.Errorf("JSON roundtrip failed for Name: expected %s, got %s", config.Name, unmarshaled.Name)
	}
	if unmarshaled.SSL != config.SSL {
		t.Errorf("JSON roundtrip failed for SSL: expected %t, got %t", config.SSL, unmarshaled.SSL)
	}
}

// TestConnectionStatusStructure tests ConnectionStatus struct
func TestConnectionStatusStructure(t *testing.T) {
	now := time.Now()
	status := &ConnectionStatus{
		Connected: true,
		LastCheck: now,
		Error:     "",
		Latency:   150,
	}

	// Test field access
	if !status.Connected {
		t.Error("Expected Connected to be true")
	}
	if status.Latency != 150 {
		t.Errorf("Expected Latency 150, got %d", status.Latency)
	}
	if status.Error != "" {
		t.Errorf("Expected empty Error, got %s", status.Error)
	}

	// Test JSON serialization
	jsonData, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("Failed to marshal ConnectionStatus to JSON: %v", err)
	}

	var unmarshaled ConnectionStatus
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal ConnectionStatus from JSON: %v", err)
	}

	if unmarshaled.Connected != status.Connected {
		t.Errorf("JSON roundtrip failed for Connected: expected %t, got %t",
			status.Connected, unmarshaled.Connected)
	}
	if unmarshaled.Latency != status.Latency {
		t.Errorf("JSON roundtrip failed for Latency: expected %d, got %d",
			status.Latency, unmarshaled.Latency)
	}
}

// TestSyncConfigStructure tests SyncConfig struct
func TestSyncConfigStructure(t *testing.T) {
	now := time.Now()
	config := &SyncConfig{
		ID:           "sync-id",
		ConnectionID: "conn-id",
		Name:         "test-sync",
		Tables: []*TableMapping{
			{
				ID:           "mapping-id",
				SyncConfigID: "sync-id",
				SourceTable:  "source_table",
				TargetTable:  "target_table",
				SyncMode:     SyncModeFull,
				Enabled:      true,
				WhereClause:  "id > 100",
				CreatedAt:    now,
				UpdatedAt:    now,
			},
		},
		SyncMode: SyncModeIncremental,
		Schedule: "0 */6 * * *",
		Enabled:  true,
		Options: &SyncOptions{
			BatchSize:          1000,
			MaxConcurrency:     4,
			EnableCompression:  true,
			ConflictResolution: ConflictResolutionOverwrite,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Test field access
	if config.ID != "sync-id" {
		t.Errorf("Expected ID 'sync-id', got %s", config.ID)
	}
	if config.ConnectionID != "conn-id" {
		t.Errorf("Expected ConnectionID 'conn-id', got %s", config.ConnectionID)
	}
	if config.SyncMode != SyncModeIncremental {
		t.Errorf("Expected SyncMode 'incremental', got %s", config.SyncMode)
	}
	if !config.Enabled {
		t.Error("Expected Enabled to be true")
	}
	if len(config.Tables) != 1 {
		t.Errorf("Expected 1 table mapping, got %d", len(config.Tables))
	}
	if config.Tables[0].SourceTable != "source_table" {
		t.Errorf("Expected SourceTable 'source_table', got %s", config.Tables[0].SourceTable)
	}
	if config.Options.BatchSize != 1000 {
		t.Errorf("Expected BatchSize 1000, got %d", config.Options.BatchSize)
	}
	if config.Options.ConflictResolution != ConflictResolutionOverwrite {
		t.Errorf("Expected ConflictResolution 'overwrite', got %s", config.Options.ConflictResolution)
	}

	// Test JSON serialization
	jsonData, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal SyncConfig to JSON: %v", err)
	}

	var unmarshaled SyncConfig
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal SyncConfig from JSON: %v", err)
	}

	if unmarshaled.ID != config.ID {
		t.Errorf("JSON roundtrip failed for ID: expected %s, got %s", config.ID, unmarshaled.ID)
	}
	if unmarshaled.SyncMode != config.SyncMode {
		t.Errorf("JSON roundtrip failed for SyncMode: expected %s, got %s",
			config.SyncMode, unmarshaled.SyncMode)
	}
}

// TestTableMappingStructure tests TableMapping struct
func TestTableMappingStructure(t *testing.T) {
	now := time.Now()
	mapping := &TableMapping{
		ID:           "mapping-id",
		SyncConfigID: "sync-id",
		SourceTable:  "users",
		TargetTable:  "local_users",
		SyncMode:     SyncModeFull,
		Enabled:      true,
		WhereClause:  "active = 1",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Test field access
	if mapping.ID != "mapping-id" {
		t.Errorf("Expected ID 'mapping-id', got %s", mapping.ID)
	}
	if mapping.SourceTable != "users" {
		t.Errorf("Expected SourceTable 'users', got %s", mapping.SourceTable)
	}
	if mapping.TargetTable != "local_users" {
		t.Errorf("Expected TargetTable 'local_users', got %s", mapping.TargetTable)
	}
	if mapping.SyncMode != SyncModeFull {
		t.Errorf("Expected SyncMode 'full', got %s", mapping.SyncMode)
	}
	if !mapping.Enabled {
		t.Error("Expected Enabled to be true")
	}
	if mapping.WhereClause != "active = 1" {
		t.Errorf("Expected WhereClause 'active = 1', got %s", mapping.WhereClause)
	}

	// Test JSON serialization
	jsonData, err := json.Marshal(mapping)
	if err != nil {
		t.Fatalf("Failed to marshal TableMapping to JSON: %v", err)
	}

	var unmarshaled TableMapping
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal TableMapping from JSON: %v", err)
	}

	if unmarshaled.SourceTable != mapping.SourceTable {
		t.Errorf("JSON roundtrip failed for SourceTable: expected %s, got %s",
			mapping.SourceTable, unmarshaled.SourceTable)
	}
	if unmarshaled.WhereClause != mapping.WhereClause {
		t.Errorf("JSON roundtrip failed for WhereClause: expected %s, got %s",
			mapping.WhereClause, unmarshaled.WhereClause)
	}
}

// TestSyncJobStructure tests SyncJob struct
func TestSyncJobStructure(t *testing.T) {
	now := time.Now()
	endTime := now.Add(time.Hour)
	job := &SyncJob{
		ID:       "job-id",
		ConfigID: "config-id",
		Status:   JobStatusRunning,
		Progress: &Progress{
			TotalTables:     5,
			CompletedTables: 2,
			TotalRows:       10000,
			ProcessedRows:   4000,
			Percentage:      40.0,
		},
		StartTime:       now,
		EndTime:         &endTime,
		TotalTables:     5,
		CompletedTables: 2,
		TotalRows:       10000,
		ProcessedRows:   4000,
		Error:           "",
		CreatedAt:       now,
	}

	// Test field access
	if job.ID != "job-id" {
		t.Errorf("Expected ID 'job-id', got %s", job.ID)
	}
	if job.Status != JobStatusRunning {
		t.Errorf("Expected Status 'running', got %s", job.Status)
	}
	if job.Progress.TotalTables != 5 {
		t.Errorf("Expected Progress.TotalTables 5, got %d", job.Progress.TotalTables)
	}
	if job.Progress.Percentage != 40.0 {
		t.Errorf("Expected Progress.Percentage 40.0, got %f", job.Progress.Percentage)
	}
	if job.TotalRows != 10000 {
		t.Errorf("Expected TotalRows 10000, got %d", job.TotalRows)
	}
	if job.EndTime == nil {
		t.Error("Expected EndTime to be set")
	}

	// Test JSON serialization
	jsonData, err := json.Marshal(job)
	if err != nil {
		t.Fatalf("Failed to marshal SyncJob to JSON: %v", err)
	}

	var unmarshaled SyncJob
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal SyncJob from JSON: %v", err)
	}

	if unmarshaled.Status != job.Status {
		t.Errorf("JSON roundtrip failed for Status: expected %s, got %s",
			job.Status, unmarshaled.Status)
	}
	if unmarshaled.TotalRows != job.TotalRows {
		t.Errorf("JSON roundtrip failed for TotalRows: expected %d, got %d",
			job.TotalRows, unmarshaled.TotalRows)
	}
}

// TestProgressStructure tests Progress struct
func TestProgressStructure(t *testing.T) {
	progress := &Progress{
		TotalTables:     10,
		CompletedTables: 7,
		TotalRows:       50000,
		ProcessedRows:   35000,
		Percentage:      70.0,
	}

	// Test field access
	if progress.TotalTables != 10 {
		t.Errorf("Expected TotalTables 10, got %d", progress.TotalTables)
	}
	if progress.CompletedTables != 7 {
		t.Errorf("Expected CompletedTables 7, got %d", progress.CompletedTables)
	}
	if progress.TotalRows != 50000 {
		t.Errorf("Expected TotalRows 50000, got %d", progress.TotalRows)
	}
	if progress.ProcessedRows != 35000 {
		t.Errorf("Expected ProcessedRows 35000, got %d", progress.ProcessedRows)
	}
	if progress.Percentage != 70.0 {
		t.Errorf("Expected Percentage 70.0, got %f", progress.Percentage)
	}

	// Test JSON serialization
	jsonData, err := json.Marshal(progress)
	if err != nil {
		t.Fatalf("Failed to marshal Progress to JSON: %v", err)
	}

	var unmarshaled Progress
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal Progress from JSON: %v", err)
	}

	if unmarshaled.Percentage != progress.Percentage {
		t.Errorf("JSON roundtrip failed for Percentage: expected %f, got %f",
			progress.Percentage, unmarshaled.Percentage)
	}
}

// TestConfigExportStructure tests ConfigExport struct
func TestConfigExportStructure(t *testing.T) {
	now := time.Now()
	export := &ConfigExport{
		Version:    "1.0",
		ExportTime: now,
		Connections: []*ConnectionConfig{
			{
				ID:          "conn-1",
				Name:        "test-connection",
				Host:        "localhost",
				Port:        3306,
				Username:    "user",
				Password:    "pass",
				Database:    "testdb",
				LocalDBName: "local_testdb",
				SSL:         false,
				CreatedAt:   now,
				UpdatedAt:   now,
			},
		},
		Mappings: []*DatabaseMapping{
			{
				RemoteConnectionID: "conn-1",
				LocalDatabaseName:  "local_testdb",
				CreatedAt:          now,
			},
		},
		SyncConfigs: []*SyncConfig{
			{
				ID:           "sync-1",
				ConnectionID: "conn-1",
				Name:         "test-sync",
				SyncMode:     SyncModeFull,
				Enabled:      true,
				CreatedAt:    now,
				UpdatedAt:    now,
			},
		},
	}

	// Test field access
	if export.Version != "1.0" {
		t.Errorf("Expected Version '1.0', got %s", export.Version)
	}
	if len(export.Connections) != 1 {
		t.Errorf("Expected 1 connection, got %d", len(export.Connections))
	}
	if len(export.Mappings) != 1 {
		t.Errorf("Expected 1 mapping, got %d", len(export.Mappings))
	}
	if len(export.SyncConfigs) != 1 {
		t.Errorf("Expected 1 sync config, got %d", len(export.SyncConfigs))
	}

	// Test JSON serialization
	jsonData, err := json.Marshal(export)
	if err != nil {
		t.Fatalf("Failed to marshal ConfigExport to JSON: %v", err)
	}

	var unmarshaled ConfigExport
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal ConfigExport from JSON: %v", err)
	}

	if unmarshaled.Version != export.Version {
		t.Errorf("JSON roundtrip failed for Version: expected %s, got %s",
			export.Version, unmarshaled.Version)
	}
	if len(unmarshaled.Connections) != len(export.Connections) {
		t.Errorf("JSON roundtrip failed for Connections length: expected %d, got %d",
			len(export.Connections), len(unmarshaled.Connections))
	}
}

// TestSyncCheckpointStructure tests SyncCheckpoint struct
func TestSyncCheckpointStructure(t *testing.T) {
	now := time.Now()
	checkpoint := &SyncCheckpoint{
		ID:             "checkpoint-id",
		TableMappingID: "mapping-id",
		LastSyncTime:   now,
		LastSyncValue:  "2023-12-01 10:00:00",
		CheckpointData: `{"last_id": 12345}`,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// Test field access
	if checkpoint.ID != "checkpoint-id" {
		t.Errorf("Expected ID 'checkpoint-id', got %s", checkpoint.ID)
	}
	if checkpoint.TableMappingID != "mapping-id" {
		t.Errorf("Expected TableMappingID 'mapping-id', got %s", checkpoint.TableMappingID)
	}
	if checkpoint.LastSyncValue != "2023-12-01 10:00:00" {
		t.Errorf("Expected LastSyncValue '2023-12-01 10:00:00', got %s", checkpoint.LastSyncValue)
	}
	if checkpoint.CheckpointData != `{"last_id": 12345}` {
		t.Errorf("Expected CheckpointData '{\"last_id\": 12345}', got %s", checkpoint.CheckpointData)
	}

	// Test JSON serialization
	jsonData, err := json.Marshal(checkpoint)
	if err != nil {
		t.Fatalf("Failed to marshal SyncCheckpoint to JSON: %v", err)
	}

	var unmarshaled SyncCheckpoint
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal SyncCheckpoint from JSON: %v", err)
	}

	if unmarshaled.LastSyncValue != checkpoint.LastSyncValue {
		t.Errorf("JSON roundtrip failed for LastSyncValue: expected %s, got %s",
			checkpoint.LastSyncValue, unmarshaled.LastSyncValue)
	}
}

// TestSyncLogStructure tests SyncLog struct
func TestSyncLogStructure(t *testing.T) {
	now := time.Now()
	log := &SyncLog{
		ID:        12345,
		JobID:     "job-id",
		TableName: "users",
		Level:     "info",
		Message:   "Successfully synced 1000 rows",
		CreatedAt: now,
	}

	// Test field access
	if log.ID != 12345 {
		t.Errorf("Expected ID 12345, got %d", log.ID)
	}
	if log.JobID != "job-id" {
		t.Errorf("Expected JobID 'job-id', got %s", log.JobID)
	}
	if log.TableName != "users" {
		t.Errorf("Expected TableName 'users', got %s", log.TableName)
	}
	if log.Level != "info" {
		t.Errorf("Expected Level 'info', got %s", log.Level)
	}
	if log.Message != "Successfully synced 1000 rows" {
		t.Errorf("Expected Message 'Successfully synced 1000 rows', got %s", log.Message)
	}

	// Test JSON serialization
	jsonData, err := json.Marshal(log)
	if err != nil {
		t.Fatalf("Failed to marshal SyncLog to JSON: %v", err)
	}

	var unmarshaled SyncLog
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal SyncLog from JSON: %v", err)
	}

	if unmarshaled.Message != log.Message {
		t.Errorf("JSON roundtrip failed for Message: expected %s, got %s",
			log.Message, unmarshaled.Message)
	}
}

// TestSyncOptionsStructure tests SyncOptions struct
func TestSyncOptionsStructure(t *testing.T) {
	options := &SyncOptions{
		BatchSize:          2000,
		MaxConcurrency:     8,
		EnableCompression:  true,
		ConflictResolution: ConflictResolutionError,
	}

	// Test field access
	if options.BatchSize != 2000 {
		t.Errorf("Expected BatchSize 2000, got %d", options.BatchSize)
	}
	if options.MaxConcurrency != 8 {
		t.Errorf("Expected MaxConcurrency 8, got %d", options.MaxConcurrency)
	}
	if !options.EnableCompression {
		t.Error("Expected EnableCompression to be true")
	}
	if options.ConflictResolution != ConflictResolutionError {
		t.Errorf("Expected ConflictResolution 'error', got %s", options.ConflictResolution)
	}

	// Test JSON serialization
	jsonData, err := json.Marshal(options)
	if err != nil {
		t.Fatalf("Failed to marshal SyncOptions to JSON: %v", err)
	}

	var unmarshaled SyncOptions
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal SyncOptions from JSON: %v", err)
	}

	if unmarshaled.ConflictResolution != options.ConflictResolution {
		t.Errorf("JSON roundtrip failed for ConflictResolution: expected %s, got %s",
			options.ConflictResolution, unmarshaled.ConflictResolution)
	}
}

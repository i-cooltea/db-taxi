# Configuration Management Implementation

## Overview

This document describes the implementation of the configuration management interface for the DB-Taxi database synchronization system. This feature allows users to export, import, validate, and backup their sync configurations.

## Features Implemented

### 1. Configuration Export
- Export all sync configurations including:
  - Connection configurations
  - Database mappings
  - Sync configurations
  - Table mappings
- Download as JSON file with timestamp
- Preview configuration before download

### 2. Configuration Import
- Upload JSON configuration file
- Automatic validation before import
- Conflict resolution options:
  - Skip conflicts (default)
  - Overwrite existing configurations
- Visual feedback for validation results

### 3. Configuration Validation
- Validates configuration file format
- Checks for required fields
- Verifies data integrity
- Provides detailed error messages

### 4. Configuration Backup
- Create backup of current configuration
- Same format as export
- Useful before making major changes

## API Endpoints

### Backend Routes (Go)

```go
// Config management routes
sync.GET("/config/export", s.exportConfig)
sync.POST("/config/import", s.importConfig)
sync.POST("/config/validate", s.validateConfig)
sync.GET("/config/backup", s.backupConfig)
```

### API Details

#### Export Configuration
- **Endpoint**: `GET /api/sync/config/export`
- **Response**: JSON configuration with all sync settings
- **Headers**: Sets Content-Disposition for file download

#### Import Configuration
- **Endpoint**: `POST /api/sync/config/import`
- **Request Body**:
  ```json
  {
    "config": { /* ConfigExport object */ },
    "resolve_conflicts": false
  }
  ```
- **Response**: Success/error message

#### Validate Configuration
- **Endpoint**: `POST /api/sync/config/validate`
- **Request Body**: ConfigExport object
- **Response**:
  ```json
  {
    "success": true,
    "valid": true,
    "message": "Configuration is valid"
  }
  ```

#### Backup Configuration
- **Endpoint**: `GET /api/sync/config/backup`
- **Response**: JSON backup with timestamp
- **Headers**: Sets Content-Disposition for file download

## Frontend Components

### ConfigManagement View (`src/views/ConfigManagement.vue`)

Main component providing the configuration management interface with:

1. **Export Section**
   - Export button to download current configuration
   - Backup button to create timestamped backup
   - Preview modal showing configuration before download

2. **Import Section**
   - File upload with drag-and-drop support
   - Validation button to check configuration
   - Visual validation feedback (success/error)
   - Conflict resolution checkbox
   - Import button (enabled after successful validation)

3. **Preview Modal**
   - Shows formatted JSON configuration
   - Download button to save to file
   - Syntax-highlighted display

### Store Methods (`src/stores/syncStore.js`)

Added methods to the Pinia store:

```javascript
// Export current configuration
async function exportConfig()

// Import configuration with optional conflict resolution
async function importConfig(configData, resolveConflicts = false)

// Validate configuration file
async function validateConfig(configData)

// Create backup of current configuration
async function backupConfig()
```

## User Interface

### Navigation
- Added "⚙️ 配置管理" link to all main views:
  - Home page
  - Connections page
  - Sync Config page
  - Monitoring page

### Visual Design
- Consistent with existing DB-Taxi design
- Purple gradient header
- Card-based layout
- Clear visual feedback for actions
- Success/error alerts
- Modal for configuration preview

## Usage Flow

### Exporting Configuration

1. Navigate to Configuration Management page
2. Click "📥 导出配置" button
3. Preview configuration in modal
4. Click "💾 下载配置文件" to save
5. File downloads with timestamp: `sync-config-YYYYMMDD-HHMMSS.json`

### Importing Configuration

1. Navigate to Configuration Management page
2. Click file upload area or drag file
3. Select JSON configuration file
4. Click "✓ 验证配置" to validate
5. Review validation results
6. (Optional) Enable "自动解决冲突" for conflict resolution
7. Click "📥 导入配置" to import
8. System refreshes with new configuration

### Creating Backup

1. Navigate to Configuration Management page
2. Click "💾 创建备份" button
3. Preview backup in modal
4. Click "💾 下载配置文件" to save
5. Backup file downloads with timestamp

## Configuration File Format

The configuration file follows the `ConfigExport` structure:

```json
{
  "version": "1.0",
  "export_time": "2026-01-11T14:00:00Z",
  "connections": [
    {
      "id": "conn-1",
      "name": "Production DB",
      "host": "db.example.com",
      "port": 3306,
      "username": "user",
      "password": "encrypted",
      "database": "mydb",
      "local_db_name": "local_mydb",
      "ssl": true
    }
  ],
  "mappings": [
    {
      "remote_connection_id": "conn-1",
      "local_database_name": "local_mydb",
      "created_at": "2026-01-11T14:00:00Z"
    }
  ],
  "sync_configs": [
    {
      "id": "sync-1",
      "connection_id": "conn-1",
      "tables": [
        {
          "source_table": "users",
          "target_table": "users",
          "sync_mode": "incremental",
          "enabled": true
        }
      ],
      "sync_mode": "incremental",
      "enabled": true
    }
  ]
}
```

## Error Handling

### Validation Errors
- Invalid JSON format
- Missing required fields
- Invalid data types
- Referential integrity issues

### Import Errors
- Configuration conflicts
- Database connection failures
- Permission issues

All errors are displayed with clear messages and suggestions for resolution.

## Security Considerations

1. **Password Handling**: Passwords in exported configurations should be handled securely
2. **File Validation**: All uploaded files are validated before processing
3. **Conflict Resolution**: Users must explicitly enable conflict resolution
4. **Backup Before Import**: Users are encouraged to backup before importing

## Testing

The implementation includes:
- Backend API endpoint tests
- Frontend component rendering
- Store method functionality
- File upload/download handling
- Validation logic

## Future Enhancements

Potential improvements:
1. Encrypted configuration exports
2. Partial configuration import (select specific items)
3. Configuration diff viewer
4. Automatic backup scheduling
5. Configuration version history
6. Import from URL
7. Configuration templates

## Requirements Validation

This implementation satisfies the following requirements:

- **Requirement 6.1**: ✅ Export configuration to file
- **Requirement 6.2**: ✅ Import and validate configuration
- **Requirement 6.3**: ✅ Display validation errors
- **Requirement 6.4**: ✅ Conflict resolution options
- **Requirement 6.5**: ✅ Automatic configuration backup

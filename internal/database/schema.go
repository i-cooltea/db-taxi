package database

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// TableInfo represents information about a database table
type TableInfo struct {
	Name        string       `json:"name"`
	Schema      string       `json:"schema"`
	Engine      string       `json:"engine"`
	Collation   string       `json:"collation"`
	Columns     []ColumnInfo `json:"columns"`
	Indexes     []IndexInfo  `json:"indexes"`
	RowCount    int64        `json:"row_count"`
	DataLength  int64        `json:"data_length"`
	IndexLength int64        `json:"index_length"`
	CreateTime  *time.Time   `json:"create_time"`
	UpdateTime  *time.Time   `json:"update_time"`
}

// ColumnInfo represents information about a table column
type ColumnInfo struct {
	Name         string  `json:"name"`
	Type         string  `json:"type"`
	Nullable     bool    `json:"nullable"`
	Key          string  `json:"key"`
	Default      *string `json:"default"`
	Extra        string  `json:"extra"`
	Comment      string  `json:"comment"`
	Position     int     `json:"position"`
	MaxLength    *int64  `json:"max_length"`
	NumericScale *int    `json:"numeric_scale"`
}

// IndexInfo represents information about a table index
type IndexInfo struct {
	Name        string   `json:"name"`
	Columns     []string `json:"columns"`
	Unique      bool     `json:"unique"`
	Type        string   `json:"type"`
	Comment     string   `json:"comment"`
	Cardinality int64    `json:"cardinality"`
}

// SchemaExplorer provides database schema exploration functionality
type SchemaExplorer struct {
	db     *sqlx.DB
	logger *logrus.Logger
}

// NewSchemaExplorer creates a new schema explorer
func NewSchemaExplorer(db *sqlx.DB, logger *logrus.Logger) *SchemaExplorer {
	if logger == nil {
		logger = logrus.New()
	}

	return &SchemaExplorer{
		db:     db,
		logger: logger,
	}
}

// GetDatabases returns a list of all databases
func (se *SchemaExplorer) GetDatabases() ([]string, error) {
	query := `
		SELECT SCHEMA_NAME 
		FROM INFORMATION_SCHEMA.SCHEMATA 
		WHERE SCHEMA_NAME NOT IN ('information_schema', 'performance_schema', 'mysql', 'sys')
		ORDER BY SCHEMA_NAME
	`

	var databases []string
	err := se.db.Select(&databases, query)
	if err != nil {
		se.logger.WithError(err).Error("Failed to get databases")
		return nil, fmt.Errorf("failed to get databases: %w", err)
	}

	se.logger.WithField("count", len(databases)).Info("Retrieved databases")
	return databases, nil
}

// GetTables returns a list of tables in the specified database
func (se *SchemaExplorer) GetTables(database string) ([]string, error) {
	if database == "" {
		return nil, fmt.Errorf("database name is required")
	}

	query := `
		SELECT TABLE_NAME 
		FROM INFORMATION_SCHEMA.TABLES 
		WHERE TABLE_SCHEMA = ? AND TABLE_TYPE = 'BASE TABLE'
		ORDER BY TABLE_NAME
	`

	var tables []string
	err := se.db.Select(&tables, query, database)
	if err != nil {
		se.logger.WithError(err).WithField("database", database).Error("Failed to get tables")
		return nil, fmt.Errorf("failed to get tables for database %s: %w", database, err)
	}

	se.logger.WithFields(logrus.Fields{
		"database": database,
		"count":    len(tables),
	}).Info("Retrieved tables")

	return tables, nil
}

// GetTableInfo returns detailed information about a specific table
func (se *SchemaExplorer) GetTableInfo(database, table string) (*TableInfo, error) {
	if database == "" || table == "" {
		return nil, fmt.Errorf("database and table names are required")
	}

	tableInfo := &TableInfo{
		Name:   table,
		Schema: database,
	}

	// Get basic table information
	if err := se.getBasicTableInfo(database, table, tableInfo); err != nil {
		return nil, err
	}

	// Get column information
	columns, err := se.getTableColumns(database, table)
	if err != nil {
		return nil, err
	}
	tableInfo.Columns = columns

	// Get index information
	indexes, err := se.getTableIndexes(database, table)
	if err != nil {
		return nil, err
	}
	tableInfo.Indexes = indexes

	se.logger.WithFields(logrus.Fields{
		"database": database,
		"table":    table,
		"columns":  len(columns),
		"indexes":  len(indexes),
		"rows":     tableInfo.RowCount,
	}).Info("Retrieved table information")

	return tableInfo, nil
}

// getBasicTableInfo retrieves basic table information
func (se *SchemaExplorer) getBasicTableInfo(database, table string, tableInfo *TableInfo) error {
	query := `
		SELECT 
			ENGINE,
			TABLE_COLLATION,
			TABLE_ROWS,
			DATA_LENGTH,
			INDEX_LENGTH,
			CREATE_TIME,
			UPDATE_TIME
		FROM INFORMATION_SCHEMA.TABLES 
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
	`

	row := se.db.QueryRow(query, database, table)
	err := row.Scan(
		&tableInfo.Engine,
		&tableInfo.Collation,
		&tableInfo.RowCount,
		&tableInfo.DataLength,
		&tableInfo.IndexLength,
		&tableInfo.CreateTime,
		&tableInfo.UpdateTime,
	)

	if err != nil {
		return fmt.Errorf("failed to get basic table info: %w", err)
	}

	return nil
}

// getTableColumns retrieves column information for a table
func (se *SchemaExplorer) getTableColumns(database, table string) ([]ColumnInfo, error) {
	query := `
		SELECT 
			COLUMN_NAME,
			COLUMN_TYPE,
			IS_NULLABLE,
			COLUMN_KEY,
			COLUMN_DEFAULT,
			EXTRA,
			COLUMN_COMMENT,
			ORDINAL_POSITION,
			CHARACTER_MAXIMUM_LENGTH,
			NUMERIC_SCALE
		FROM INFORMATION_SCHEMA.COLUMNS 
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		ORDER BY ORDINAL_POSITION
	`

	rows, err := se.db.Query(query, database, table)
	if err != nil {
		return nil, fmt.Errorf("failed to query table columns: %w", err)
	}
	defer rows.Close()

	var columns []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		var nullable string

		err := rows.Scan(
			&col.Name,
			&col.Type,
			&nullable,
			&col.Key,
			&col.Default,
			&col.Extra,
			&col.Comment,
			&col.Position,
			&col.MaxLength,
			&col.NumericScale,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan column: %w", err)
		}

		col.Nullable = nullable == "YES"
		columns = append(columns, col)
	}

	return columns, nil
}

// getTableIndexes retrieves index information for a table
func (se *SchemaExplorer) getTableIndexes(database, table string) ([]IndexInfo, error) {
	query := `
		SELECT 
			INDEX_NAME,
			COLUMN_NAME,
			NON_UNIQUE,
			INDEX_TYPE,
			INDEX_COMMENT,
			CARDINALITY,
			SEQ_IN_INDEX
		FROM INFORMATION_SCHEMA.STATISTICS 
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		ORDER BY INDEX_NAME, SEQ_IN_INDEX
	`

	rows, err := se.db.Query(query, database, table)
	if err != nil {
		return nil, fmt.Errorf("failed to query table indexes: %w", err)
	}
	defer rows.Close()

	indexMap := make(map[string]*IndexInfo)
	for rows.Next() {
		var indexName, columnName, indexType, comment string
		var nonUnique, cardinality int64
		var seqInIndex int

		err := rows.Scan(
			&indexName,
			&columnName,
			&nonUnique,
			&indexType,
			&comment,
			&cardinality,
			&seqInIndex,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan index: %w", err)
		}

		if index, exists := indexMap[indexName]; exists {
			index.Columns = append(index.Columns, columnName)
		} else {
			indexMap[indexName] = &IndexInfo{
				Name:        indexName,
				Columns:     []string{columnName},
				Unique:      nonUnique == 0,
				Type:        indexType,
				Comment:     comment,
				Cardinality: cardinality,
			}
		}
	}

	var indexes []IndexInfo
	for _, index := range indexMap {
		indexes = append(indexes, *index)
	}

	return indexes, nil
}

// GetTableData retrieves data from a specific table with pagination
func (se *SchemaExplorer) GetTableData(database, table string, offset, limit int) ([]map[string]interface{}, error) {
	if database == "" || table == "" {
		return nil, fmt.Errorf("database and table names are required")
	}

	if limit <= 0 {
		limit = 100 // Default limit
	}
	if limit > 1000 {
		limit = 1000 // Maximum limit
	}

	// Build query with proper escaping
	query := fmt.Sprintf("SELECT * FROM `%s`.`%s` LIMIT ? OFFSET ?", database, table)

	rows, err := se.db.Query(query, limit, offset)
	if err != nil {
		se.logger.WithError(err).WithFields(logrus.Fields{
			"database": database,
			"table":    table,
			"limit":    limit,
			"offset":   offset,
		}).Error("Failed to query table data")
		return nil, fmt.Errorf("failed to query table data: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	var results []map[string]interface{}
	for rows.Next() {
		// Create a slice of interface{} to hold the values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// Scan the row
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Create a map for this row
		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if val != nil {
				// Convert byte arrays to strings for better JSON serialization
				if b, ok := val.([]byte); ok {
					row[col] = string(b)
				} else {
					row[col] = val
				}
			} else {
				row[col] = nil
			}
		}
		results = append(results, row)
	}

	se.logger.WithFields(logrus.Fields{
		"database": database,
		"table":    table,
		"rows":     len(results),
		"limit":    limit,
		"offset":   offset,
	}).Info("Retrieved table data")

	return results, nil
}

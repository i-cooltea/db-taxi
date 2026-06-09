package sync

import (
	"context"
	"crypto/md5"
	"database/sql"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// DiffResult is the top-level result of a table diff operation.
type DiffResult struct {
	SchemaDiff *SchemaDiff `json:"schema_diff"`
	DataDiff   *DataDiff   `json:"data_diff"`
	Warning    string      `json:"warning,omitempty"`
}

// SchemaDiff describes column-level differences between two tables.
type SchemaDiff struct {
	AddedColumns    []ColumnDiff `json:"added_columns"`
	RemovedColumns  []ColumnDiff `json:"removed_columns"`
	ModifiedColumns []ColumnDiff `json:"modified_columns"`
	IsSame          bool         `json:"is_same"`
}

// ColumnDiff represents a single column difference.
type ColumnDiff struct {
	Name         string `json:"name"`
	Type         string `json:"type,omitempty"`
	Nullable     bool   `json:"nullable,omitempty"`
	DefaultValue string `json:"default_value,omitempty"`
	// Only for modified columns
	SourceType     string `json:"source_type,omitempty"`
	TargetType     string `json:"target_type,omitempty"`
	SourceNullable bool   `json:"source_nullable,omitempty"`
	TargetNullable bool   `json:"target_nullable,omitempty"`
}

// DataDiff describes row-level differences between two tables.
type DataDiff struct {
	OnlyInSource []map[string]interface{} `json:"only_in_source"`
	OnlyInTarget []map[string]interface{} `json:"only_in_target"`
	ValueDiff    []ValueDiffEntry         `json:"value_diff"`
	PrimaryKey   []string                 `json:"primary_key"`
	Partial      bool                     `json:"partial"`
	TotalRows    int64                    `json:"total_rows"`
	ComparedRows int64                    `json:"compared_rows"`
}

// ValueDiffEntry represents a row where values differ between source and target.
type ValueDiffEntry struct {
	PrimaryKey  map[string]interface{} `json:"primary_key"`
	Differences []ColumnValueDiff      `json:"differences"`
	SourceRow   map[string]interface{} `json:"source_row"`
}

// ColumnValueDiff represents a single column value difference within a row.
type ColumnValueDiff struct {
	Column      string      `json:"column"`
	SourceValue interface{} `json:"source_value"`
	TargetValue interface{} `json:"target_value"`
}

// columnRow mirrors a row from INFORMATION_SCHEMA.COLUMNS.
type columnRow struct {
	ColumnName      string         `db:"COLUMN_NAME"`
	ColumnType      string         `db:"COLUMN_TYPE"`
	IsNullable      string         `db:"IS_NULLABLE"`
	ColumnDefault   sql.NullString `db:"COLUMN_DEFAULT"`
	OrdinalPosition int            `db:"ORDINAL_POSITION"`
}

// DiffEngine performs table diff operations.
type DiffEngine struct {
	repo   Repository
	logger *logrus.Logger
}

// NewDiffEngine creates a new DiffEngine.
func NewDiffEngine(repo Repository, logger *logrus.Logger) *DiffEngine {
	return &DiffEngine{repo: repo, logger: logger}
}

// RunDiff orchestrates the full diff: schema + data.
func (e *DiffEngine) RunDiff(ctx context.Context, sourceDB, targetDB *sqlx.DB, sourceDBName, targetDBName, sourceTable, targetTable string, maxRows int64) (*DiffResult, error) {
	result := &DiffResult{}

	// 1. Schema diff
	schemaDiff, err := e.CompareSchema(ctx, sourceDB, targetDB, sourceDBName, targetDBName, sourceTable, targetTable)
	if err != nil {
		return nil, fmt.Errorf("schema diff failed: %w", err)
	}
	result.SchemaDiff = schemaDiff

	// 2. Data diff — requires primary key
	sourcePK, err := e.GetPrimaryKeys(ctx, sourceDB, sourceDBName, sourceTable)
	if err != nil {
		result.Warning = fmt.Sprintf("源表无法获取主键: %v，跳过数据差异对比", err)
		result.DataDiff = &DataDiff{PrimaryKey: nil}
		return result, nil
	}

	targetPK, err := e.GetPrimaryKeys(ctx, targetDB, targetDBName, targetTable)
	if err != nil {
		result.Warning = fmt.Sprintf("目标表无法获取主键: %v，跳过数据差异对比", err)
		result.DataDiff = &DataDiff{PrimaryKey: nil}
		return result, nil
	}

	// Check PK compatibility
	if !pkEqual(sourcePK, targetPK) {
		result.Warning = "源表和目标表主键列不一致，跳过数据差异对比"
		result.DataDiff = &DataDiff{PrimaryKey: sourcePK}
		return result, nil
	}

	dataDiff, err := e.CompareData(ctx, sourceDB, targetDB, sourceDBName, targetDBName, sourceTable, targetTable, sourcePK, maxRows)
	if err != nil {
		return nil, fmt.Errorf("data diff failed: %w", err)
	}
	result.DataDiff = dataDiff

	return result, nil
}

// CompareSchema compares column structures of two tables.
func (e *DiffEngine) CompareSchema(ctx context.Context, sourceDB, targetDB *sqlx.DB, sourceDBName, targetDBName, sourceTable, targetTable string) (*SchemaDiff, error) {
	sourceCols, err := e.fetchColumns(ctx, sourceDB, sourceDBName, sourceTable)
	if err != nil {
		return nil, fmt.Errorf("fetch source columns: %w", err)
	}
	targetCols, err := e.fetchColumns(ctx, targetDB, targetDBName, targetTable)
	if err != nil {
		return nil, fmt.Errorf("fetch target columns: %w", err)
	}

	diff := &SchemaDiff{IsSame: true}

	sourceMap := colMap(sourceCols)
	targetMap := colMap(targetCols)

	// Added columns (in source but not in target)
	for name, col := range sourceMap {
		if _, exists := targetMap[name]; !exists {
			diff.AddedColumns = append(diff.AddedColumns, ColumnDiff{
				Name:         name,
				Type:         col.ColumnType,
				Nullable:     col.IsNullable == "YES",
				DefaultValue: col.ColumnDefault.String,
			})
			diff.IsSame = false
		}
	}

	// Removed columns (in target but not in source)
	for name, col := range targetMap {
		if _, exists := sourceMap[name]; !exists {
			diff.RemovedColumns = append(diff.RemovedColumns, ColumnDiff{
				Name:         name,
				Type:         col.ColumnType,
				Nullable:     col.IsNullable == "YES",
				DefaultValue: col.ColumnDefault.String,
			})
			diff.IsSame = false
		}
	}

	// Modified columns (exist in both but differ)
	for name, srcCol := range sourceMap {
		tgtCol, exists := targetMap[name]
		if !exists {
			continue
		}
		if srcCol.ColumnType != tgtCol.ColumnType || srcCol.IsNullable != tgtCol.IsNullable {
			diff.ModifiedColumns = append(diff.ModifiedColumns, ColumnDiff{
				Name:           name,
				SourceType:     srcCol.ColumnType,
				TargetType:     tgtCol.ColumnType,
				SourceNullable: srcCol.IsNullable == "YES",
				TargetNullable: tgtCol.IsNullable == "YES",
			})
			diff.IsSame = false
		}
	}

	// Sort for deterministic output
	sort.Slice(diff.AddedColumns, func(i, j int) bool { return diff.AddedColumns[i].Name < diff.AddedColumns[j].Name })
	sort.Slice(diff.RemovedColumns, func(i, j int) bool { return diff.RemovedColumns[i].Name < diff.RemovedColumns[j].Name })
	sort.Slice(diff.ModifiedColumns, func(i, j int) bool { return diff.ModifiedColumns[i].Name < diff.ModifiedColumns[j].Name })

	return diff, nil
}

// GetPrimaryKeys returns the primary key column names for a table.
func (e *DiffEngine) GetPrimaryKeys(ctx context.Context, db *sqlx.DB, dbName, tableName string) ([]string, error) {
	query := `
		SELECT COLUMN_NAME
		FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? AND CONSTRAINT_NAME = 'PRIMARY'
		ORDER BY ORDINAL_POSITION
	`
	var pks []string
	err := db.SelectContext(ctx, &pks, query, dbName, tableName)
	if err != nil {
		return nil, fmt.Errorf("query primary keys: %w", err)
	}
	if len(pks) == 0 {
		return nil, fmt.Errorf("表 %s.%s 没有主键", dbName, tableName)
	}
	return pks, nil
}

// CompareData compares row data between two tables based on primary key matching.
func (e *DiffEngine) CompareData(ctx context.Context, sourceDB, targetDB *sqlx.DB, sourceDBName, targetDBName, sourceTable, targetTable string, primaryKeys []string, maxRows int64) (*DataDiff, error) {
	diff := &DataDiff{
		PrimaryKey: primaryKeys,
	}

	// Get total row count from source
	var totalRows int64
	err := sourceDB.GetContext(ctx, &totalRows, fmt.Sprintf("SELECT COUNT(*) FROM `%s`.`%s`", sourceDBName, sourceTable))
	if err != nil {
		return nil, fmt.Errorf("count source rows: %w", err)
	}
	diff.TotalRows = totalRows

	// Determine effective limit
	effectiveLimit := totalRows
	if maxRows > 0 && totalRows > maxRows {
		effectiveLimit = maxRows
		diff.Partial = true
	}
	diff.ComparedRows = effectiveLimit

	pkCols := strings.Join(primaryKeys, "`,`")

	// 1. Fetch all rows (up to limit) from both tables
	sourceRows, err := e.fetchRows(ctx, sourceDB, sourceDBName, sourceTable, pkCols, effectiveLimit)
	if err != nil {
		return nil, fmt.Errorf("fetch source rows: %w", err)
	}
	targetRows, err := e.fetchRows(ctx, targetDB, targetDBName, targetTable, pkCols, effectiveLimit)
	if err != nil {
		return nil, fmt.Errorf("fetch target rows: %w", err)
	}

	// 2. Index target rows by PK for fast lookup
	targetByPK := make(map[string]map[string]interface{})
	for _, row := range targetRows {
		pk := pkValue(primaryKeys, row)
		targetByPK[pk] = row
	}

	// 3. Diff
	sourceByPK := make(map[string]map[string]interface{})
	for _, row := range sourceRows {
		pk := pkValue(primaryKeys, row)
		sourceByPK[pk] = row

		tgtRow, exists := targetByPK[pk]
		if !exists {
			diff.OnlyInSource = append(diff.OnlyInSource, row)
			continue
		}
		// Compare values
		var diffs []ColumnValueDiff
		for col, srcVal := range row {
			tgtVal, hasCol := tgtRow[col]
			if !hasCol {
				continue
			}
			if fmt.Sprintf("%v", srcVal) != fmt.Sprintf("%v", tgtVal) {
				diffs = append(diffs, ColumnValueDiff{
					Column:      col,
					SourceValue: srcVal,
					TargetValue: tgtVal,
				})
			}
		}
		if len(diffs) > 0 {
			sort.Slice(diffs, func(i, j int) bool { return diffs[i].Column < diffs[j].Column })
			diff.ValueDiff = append(diff.ValueDiff, ValueDiffEntry{
				PrimaryKey:  pkMap(primaryKeys, row),
				Differences: diffs,
				SourceRow:   row,
			})
		}
	}

	// Rows only in target
	for pk, row := range targetByPK {
		if _, exists := sourceByPK[pk]; !exists {
			diff.OnlyInTarget = append(diff.OnlyInTarget, row)
		}
	}

	return diff, nil
}

// --- helpers ---

func (e *DiffEngine) fetchColumns(ctx context.Context, db *sqlx.DB, dbName, tableName string) ([]columnRow, error) {
	query := `
		SELECT COLUMN_NAME, COLUMN_TYPE, IS_NULLABLE, COLUMN_DEFAULT, ORDINAL_POSITION
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		ORDER BY ORDINAL_POSITION
	`
	var cols []columnRow
	err := db.SelectContext(ctx, &cols, query, dbName, tableName)
	if err != nil {
		return nil, err
	}
	return cols, nil
}

func (e *DiffEngine) fetchRows(ctx context.Context, db *sqlx.DB, dbName, tableName, pkCols string, limit int64) ([]map[string]interface{}, error) {
	query := fmt.Sprintf("SELECT * FROM `%s`.`%s` ORDER BY `%s` LIMIT %d", dbName, tableName, pkCols, limit)
	rows, err := db.QueryxContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		row := make(map[string]interface{})
		if err := rows.MapScan(row); err != nil {
			return nil, err
		}
		// Convert []byte to string for JSON serialization
		for k, v := range row {
			if b, ok := v.([]byte); ok {
				row[k] = string(b)
			}
		}
		results = append(results, row)
	}
	return results, nil
}

func colMap(cols []columnRow) map[string]columnRow {
	m := make(map[string]columnRow, len(cols))
	for _, c := range cols {
		m[c.ColumnName] = c
	}
	return m
}

func pkEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func pkValue(pks []string, row map[string]interface{}) string {
	vals := make([]string, len(pks))
	for i, pk := range pks {
		vals[i] = fmt.Sprintf("%v", row[pk])
	}
	return strings.Join(vals, "|")
}

func pkMap(pks []string, row map[string]interface{}) map[string]interface{} {
	m := make(map[string]interface{}, len(pks))
	for _, pk := range pks {
		m[pk] = row[pk]
	}
	return m
}

// computeRowHash returns an MD5 hash of all values in a row (used for fast comparison).
func computeRowHash(row map[string]interface{}) string {
	keys := make([]string, 0, len(row))
	for k := range row {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf strings.Builder
	for _, k := range keys {
		buf.WriteString(k)
		buf.WriteByte('=')
		buf.WriteString(fmt.Sprintf("%v", row[k]))
		buf.WriteByte(';')
	}
	return fmt.Sprintf("%x", md5.Sum([]byte(buf.String())))
}

// ValidateDiffTableRequest checks that both tables exist.
func (e *DiffEngine) ValidateDiffTableRequest(ctx context.Context, db *sqlx.DB, dbName, tableName string) (int, error) {
	var count int
	err := db.GetContext(ctx, &count,
		"SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?",
		dbName, tableName,
	)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("check table existence: %w", err)
	}
	if count == 0 {
		return http.StatusBadRequest, fmt.Errorf("表 %s.%s 不存在", dbName, tableName)
	}
	return 0, nil
}

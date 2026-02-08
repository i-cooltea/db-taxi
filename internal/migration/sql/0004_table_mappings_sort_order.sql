-- Version: 4
-- Name: table_mappings_sort_order
-- Description: Add sort_order to table_mappings for custom sync order
ALTER TABLE `table_mappings`
ADD COLUMN `sort_order` INT NOT NULL DEFAULT 0 AFTER `where_clause`;

CREATE INDEX `idx_table_mappings_sort_order` ON `table_mappings` (`sync_config_id`, `sort_order`);

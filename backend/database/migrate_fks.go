package database

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"gorm.io/gorm"
)

// fkConstraint declares one real FOREIGN KEY to enforce (WP-4.1 C4).
type fkConstraint struct {
	Table      string
	Column     string
	References string
	RefColumn  string
	OnDelete   string
}

// fkConstraints lists every constraint the migration enforces.
//
// Safety contract with the erasure map (privacy_repo.DeleteAccountTx):
// CASCADE is used ONLY on tables whose rows are DELETEd during account
// erasure (learner-private data). Tables whose user reference is ANONYMIZED
// to "" (audit_logs.user_id, announcements.author_id, assignments.created_by,
// classes.teacher_id) deliberately get NO constraint — a FK would reject the
// blanking. Optional/unset references (assignments.activity_id, OTP phone
// keys) get none either, so "" values never violate integrity.
var fkConstraints = []fkConstraint{
	{"learner_activities", "learner_id", "users", "id", "CASCADE"},
	{"learner_activities", "activity_id", "activities", "id", "CASCADE"},
	{"progresses", "learner_id", "users", "id", "CASCADE"},
	{"observations", "learner_id", "users", "id", "CASCADE"},
	{"guidances", "learner_id", "users", "id", "CASCADE"},
	{"daily_activities", "learner_id", "users", "id", "CASCADE"},
	{"enrollments", "user_id", "users", "id", "CASCADE"},
	{"enrollments", "course_id", "courses", "id", "CASCADE"},
	{"micro_modules", "activity_id", "activities", "id", "CASCADE"},
	{"class_members", "class_id", "classes", "id", "CASCADE"},
	{"class_members", "user_id", "users", "id", "CASCADE"},
	{"submissions", "learner_id", "users", "id", "CASCADE"},
	{"submissions", "assignment_id", "assignments", "id", "CASCADE"},
	{"token_blocklists", "user_id", "users", "id", "CASCADE"},
	{"user_revocations", "user_id", "users", "id", "CASCADE"},
	{"consent_records", "user_id", "users", "id", "CASCADE"},
	// parent_links.parent_id deliberately gets NO constraint: a pending
	// invite (ParentLinkStatusPending) is created BEFORE the guardian exists,
	// so the column legitimately holds "" until claim. student_id is always a
	// real enrolled learner, so it is enforced. Account erasure still deletes
	// links by parent_id (privacy_repo.DeleteAccountTx).
	{"parent_links", "student_id", "users", "id", "CASCADE"},
	{"support_issues", "user_id", "users", "id", "CASCADE"},
	{"learner_notes", "student_id", "users", "id", "CASCADE"},
}

// MigrateForeignKeys (WP-4.1 C4) adds real, enforced FOREIGN KEY constraints
// to tables that gorm.AutoMigrate created without them (SQLite cannot
// ALTER TABLE ADD CONSTRAINT, so each affected table is rebuilt via the
// classic create-new → copy → drop-old → rename recipe, with its indexes
// recreated under their original names).
//
// Idempotent: constraints already present (PRAGMA foreign_key_list) are
// skipped. Non-destructive: a table with orphan rows (references to missing
// parents) is skipped with a warning — data is never deleted to fit a
// constraint; the constraint lands once the data is clean. On a stock
// database the erasure map (which now covers every CASCADE table) guarantees
// no orphans exist.
func MigrateForeignKeys(db *gorm.DB) {
	for _, fc := range fkConstraints {
		if !tableExists(db, fc.Table) || !tableExists(db, fc.References) {
			continue // table not created yet (older DB); AutoMigrate will catch up
		}
		if hasForeignKey(db, fc.Table, fc.Column, fc.References) {
			continue // already enforced
		}
		if orphans := orphanCount(db, fc); orphans > 0 {
			slog.Warn("FK migration skipped — orphan rows must be cleaned first (data preserved)",
				"table", fc.Table, "column", fc.Column, "references", fc.References, "orphans", orphans)
			continue
		}
		if err := rebuildTableWithFK(db, fc); err != nil {
			slog.Error("FK migration failed",
				"table", fc.Table, "column", fc.Column, "references", fc.References, "error", err)
		}
	}
}

type columnDef struct {
	name     string
	typ      string
	notNull  bool
	defValue *string
	pkOrder  int // 0 = not part of the primary key
}

func tableColumns(db *gorm.DB, table string) ([]columnDef, error) {
	var infos []struct {
		Cid      int     `gorm:"column:cid"`
		Name     string  `gorm:"column:name"`
		Type     string  `gorm:"column:type"`
		NotNull  int     `gorm:"column:notnull"`
		DefValue *string `gorm:"column:dflt_value"`
		PK       int     `gorm:"column:pk"`
	}
	if err := db.Raw("PRAGMA table_info(" + table + ")").Scan(&infos).Error; err != nil {
		return nil, err
	}
	cols := make([]columnDef, 0, len(infos))
	for _, i := range infos {
		cols = append(cols, columnDef{name: i.Name, typ: i.Type, notNull: i.NotNull == 1, defValue: i.DefValue, pkOrder: i.PK})
	}
	sort.SliceStable(cols, func(a, b int) bool {
		return cols[a].pkOrder > 0 && (cols[b].pkOrder == 0 || cols[a].pkOrder < cols[b].pkOrder)
	})
	return cols, nil
}

type indexDef struct {
	name    string
	unique  bool
	columns []string
}

// fkRef is one existing FOREIGN KEY read from PRAGMA foreign_key_list, so a
// rebuild preserves constraints already on the table alongside the new one
// (a single rebuild must never silently drop sibling FKs).
type fkRef struct {
	from     string
	table    string
	to       string
	onDelete string
}

func existingFKs(db *gorm.DB, table string) []fkRef {
	var rows []struct {
		ID       int    `gorm:"column:id"`
		Table    string `gorm:"column:table"`
		From     string `gorm:"column:from"`
		To       string `gorm:"column:to"`
		OnDelete string `gorm:"column:on_delete"`
	}
	if err := db.Raw("PRAGMA foreign_key_list(" + table + ")").Scan(&rows).Error; err != nil {
		return nil
	}
	var fks []fkRef
	seen := map[string]bool{}
	for _, r := range rows {
		key := r.ID
		if seen[fmt.Sprintf("%d", key)] {
			continue
		}
		seen[fmt.Sprintf("%d", key)] = true
		fks = append(fks, fkRef{from: r.From, table: r.Table, to: r.To, onDelete: r.OnDelete})
	}
	return fks
}

func tableIndexes(db *gorm.DB, table string) ([]indexDef, error) {
	var list []struct {
		Seq     int    `gorm:"column:seq"`
		Name    string `gorm:"column:name"`
		Unique  int    `gorm:"column:unique"`
		Origin  string `gorm:"column:origin"`
		Partial int    `gorm:"column:partial"`
	}
	if err := db.Raw("PRAGMA index_list(" + table + ")").Scan(&list).Error; err != nil {
		return nil, err
	}
	var indexes []indexDef
	for _, l := range list {
		if l.Origin == "pk" {
			continue // part of the table definition
		}
		var cols []struct {
			Seqno int    `gorm:"column:seqno"`
			Name  string `gorm:"column:name"`
		}
		if err := db.Raw("PRAGMA index_info(" + l.Name + ")").Scan(&cols).Error; err != nil {
			return nil, err
		}
		names := make([]string, 0, len(cols))
		for _, c := range cols {
			names = append(names, c.Name)
		}
		indexes = append(indexes, indexDef{name: l.Name, unique: l.Unique == 1, columns: names})
	}
	return indexes, nil
}

func hasForeignKey(db *gorm.DB, table, column, refTable string) bool {
	var fks []struct {
		Table string `gorm:"column:table"`
		From  string `gorm:"column:from"`
		To    string `gorm:"column:to"`
	}
	if err := db.Raw("PRAGMA foreign_key_list(" + table + ")").Scan(&fks).Error; err != nil {
		return false
	}
	for _, fk := range fks {
		if fk.From == column && fk.Table == refTable && fk.To == "id" {
			return true
		}
	}
	return false
}

func orphanCount(db *gorm.DB, fc fkConstraint) int64 {
	var count int64
	db.Raw(fmt.Sprintf(
		`SELECT COUNT(*) FROM %q t WHERE t.%q IS NOT NULL AND NOT EXISTS (SELECT 1 FROM %q r WHERE r.%q = t.%q)`,
		fc.Table, fc.Column, fc.References, fc.RefColumn, fc.Column)).Scan(&count)
	return count
}

// rebuildTableWithFK recreates the table with the constraint applied. The
// recipe only rebuilds CHILD tables (their parents are never dropped here),
// so no FK toggle is needed: the copy INSERT validates against parents that
// exist, and dropping the old child table removes its constraints with it.
func rebuildTableWithFK(db *gorm.DB, fc fkConstraint) error {
	cols, err := tableColumns(db, fc.Table)
	if err != nil {
		return err
	}
	indexes, err := tableIndexes(db, fc.Table)
	if err != nil {
		return err
	}

	defs := make([]string, 0, len(cols)+2)
	quotedNames := make([]string, 0, len(cols))
	for _, c := range cols {
		quotedNames = append(quotedNames, `"`+c.name+`"`)
		ddl := `"` + c.name + `" ` + c.typ
		if c.notNull {
			ddl += " NOT NULL"
		}
		if c.defValue != nil {
			ddl += " DEFAULT " + *c.defValue
		}
		defs = append(defs, ddl)
	}
	var pkCols []string
	for _, c := range cols {
		if c.pkOrder > 0 {
			pkCols = append(pkCols, `"`+c.name+`"`)
		}
	}
	if len(pkCols) > 0 {
		defs = append(defs, "PRIMARY KEY ("+strings.Join(pkCols, ", ")+")")
	}
	// Preserve any constraints already on the table, then append the new one.
	for _, fk := range existingFKs(db, fc.Table) {
		defs = append(defs, fmt.Sprintf("FOREIGN KEY (%q) REFERENCES %q(%q) ON DELETE %s", fk.from, fk.table, fk.to, fk.onDelete))
	}
	defs = append(defs, fmt.Sprintf("FOREIGN KEY (%q) REFERENCES %q(%q) ON DELETE %s", fc.Column, fc.References, fc.RefColumn, fc.OnDelete))

	newTable := "new_" + fc.Table
	createDDL := "CREATE TABLE " + `"` + newTable + `"` + " (" + strings.Join(defs, ", ") + ")"
	copyDDL := fmt.Sprintf("INSERT INTO %q SELECT %s FROM %q", newTable, strings.Join(quotedNames, ", "), fc.Table)

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(createDDL).Error; err != nil {
			return err
		}
		if err := tx.Exec(copyDDL).Error; err != nil {
			return err
		}
		if err := tx.Exec(`DROP TABLE "` + fc.Table + `"`).Error; err != nil {
			return err
		}
		if err := tx.Exec(`ALTER TABLE "` + newTable + `" RENAME TO "` + fc.Table + `"`).Error; err != nil {
			return err
		}
		for _, idx := range indexes {
			cols := make([]string, 0, len(idx.columns))
			for _, c := range idx.columns {
				cols = append(cols, `"`+c+`"`)
			}
			unique := ""
			if idx.unique {
				unique = " UNIQUE"
			}
			if err := tx.Exec(fmt.Sprintf("CREATE%s INDEX IF NOT EXISTS %q ON %q(%s)", unique, idx.name, fc.Table, strings.Join(cols, ", "))).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func tableExists(db *gorm.DB, name string) bool {
	var count int64
	db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?", name).Scan(&count)
	return count > 0
}

// Command migrate 按顺序执行尚未应用的 SQL 文件，并将文件名记录到
// schema_migrations 表中，以支持可重复部署。
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	dir := flag.String("dir", "migrations", "migration directory")
	flag.Parse()
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		exit("MYSQL_DSN is required")
	}
	db, e := sql.Open("mysql", dsn)
	if e != nil {
		exit(e.Error())
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if _, e = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations(version VARCHAR(100) PRIMARY KEY,applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP) ENGINE=InnoDB`); e != nil {
		exit(e.Error())
	}
	files, e := filepath.Glob(filepath.Join(*dir, "*.sql"))
	if e != nil {
		exit(e.Error())
	}
	// 通过 001_、002_ 等文件名字典序确定迁移执行顺序。
	sort.Strings(files)
	for _, p := range files {
		name := filepath.Base(p)
		var n int
		if e = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version=?`, name).Scan(&n); e != nil {
			exit(e.Error())
		}
		if n > 0 {
			continue
		}
		b, e := os.ReadFile(p)
		if e != nil {
			exit(e.Error())
		}
		tx, e := db.BeginTx(ctx, nil)
		if e != nil {
			exit(e.Error())
		}
		ok := true
		for _, q := range splitSQL(string(b)) {
			if _, e = tx.ExecContext(ctx, q); e != nil {
				ok = false
				break
			}
		}
		if ok {
			_, e = tx.ExecContext(ctx, `INSERT INTO schema_migrations(version) VALUES(?)`, name)
		}
		if e != nil || !ok {
			tx.Rollback()
			exit(fmt.Sprintf("migration %s: %v", name, e))
		}
		if e = tx.Commit(); e != nil {
			exit(e.Error())
		}
		fmt.Println("applied", name)
	}
}

// splitSQL 支持本项目使用的 DDL/DML 迁移文件。包含内部分号的存储过程
// 不属于此轻量迁移工具的支持范围。
func splitSQL(s string) []string {
	lines := strings.Split(s, "\n")
	clean := make([]string, 0, len(lines))
	for _, l := range lines {
		if !strings.HasPrefix(strings.TrimSpace(l), "--") {
			clean = append(clean, l)
		}
	}
	parts := strings.Split(strings.Join(clean, "\n"), ";")
	out := []string{}
	for _, p := range parts {
		if q := strings.TrimSpace(p); q != "" {
			out = append(out, q)
		}
	}
	return out
}
func exit(s string) { fmt.Fprintln(os.Stderr, s); os.Exit(1) }

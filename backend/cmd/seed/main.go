package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"supervision-system/internal/security"
)

func main() {
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		panic("MYSQL_DSN is required")
	}
	db, e := sql.Open("mysql", dsn)
	if e != nil {
		panic(e)
	}
	defer db.Close()
	accounts := []struct{ Username, Name, Role, DeptCode string }{{"leader", "王书记", "leader", ""}, {"office", "办公室联络员", "department", "office"}, {"political", "政治处联络员", "department", "political"}, {"admin", "系统管理员", "admin", ""}}
	h, e := security.HashPassword("12345678")
	if e != nil {
		panic(e)
	}
	for _, a := range accounts {
		var dep any = nil
		if a.DeptCode != "" {
			var id int64
			if e = db.QueryRowContext(context.Background(), `SELECT id FROM departments WHERE code=?`, a.DeptCode).Scan(&id); e != nil {
				panic(e)
			}
			dep = id
		}
		_, e = db.Exec(`INSERT INTO users(username,name,password_hash,role,department_id,enabled) VALUES(?,?,?,?,?,1) ON DUPLICATE KEY UPDATE name=VALUES(name),password_hash=VALUES(password_hash),role=VALUES(role),department_id=VALUES(department_id),enabled=1`, a.Username, a.Name, h, a.Role, dep)
		if e != nil {
			panic(e)
		}
	}
	fmt.Println("demo accounts seeded; password: 12345678")
}

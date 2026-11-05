// Command bootstrap 仅供部署初始化使用：创建管理员，不修改已有账号。
package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"supervision-system/internal/security"
)

func main() {
	dsn, password := os.Getenv("MYSQL_DSN"), os.Getenv("BOOTSTRAP_ADMIN_PASSWORD")
	if dsn == "" || len(password) < 12 || len(password) > 72 {
		log.Fatal("必须配置 MYSQL_DSN 和 12~72 字节的 BOOTSTRAP_ADMIN_PASSWORD")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	hash, err := security.HashPassword(password)
	if err != nil {
		log.Fatal(err)
	}
	// 唯一键冲突时不覆盖密码、角色和启用状态，容器重启不会重置账号。
	_, err = db.ExecContext(ctx, `INSERT INTO users(username,name,password_hash,role,enabled)
		VALUES('admin','系统管理员',?,'admin',1) ON DUPLICATE KEY UPDATE id=id`, hash)
	if err != nil {
		log.Fatal(err)
	}
	log.Print("管理员初始化完成；已有账号保持不变")
}

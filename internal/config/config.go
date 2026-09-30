package config

import (
	"flag"
	"os"
	"strconv"
)

// Config 保存服务运行所需的所有配置。
type Config struct {
	// Addr HTTP listen address
	Addr string
	// DSN 数据库连接串
	DSN string
	// WebDir 静态页面目录
	WebDir string
	// UploadDir 上传目录
	UploadDir string
	// DefaultAdminUser 初始化管理员账号
	DefaultAdminUser string
	// DefaultAdminPass 初始化管理员密码
	DefaultAdminPass string
	// AutoMigrate auto-create tables on start
	AutoMigrate bool
	// Seed seed sample data on start
	Seed bool
}

// MySQL 连接参数
type dbOptions struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// Load 解析命令行与环境变量，生成配置。
func Load() *Config {
	c := &Config{}
	fs := flag.NewFlagSet("vsite", flag.ExitOnError)

	dbHost := fs.String("db-host", env("DB_HOST", "localhost"), "MySQL host")
	dbPort := fs.Int("db-port", 3306, "MySQL port")
	dbUser := fs.String("db-user", env("DB_USER", "admin"), "MySQL user")
	dbPass := fs.String("db-pass", env("DB_PASS", "!QAZ2wsx"), "MySQL password")
	dbName := fs.String("db-name", env("DB_NAME", "vsite"), "MySQL database name")

	fs.StringVar(&c.Addr, "addr", env("ADDR", ":8080"), "HTTP listen address")
	fs.StringVar(&c.WebDir, "web", env("WEB_DIR", "./web"), "web pages directory")
	fs.StringVar(&c.UploadDir, "upload", env("UPLOAD_DIR", "./web/static/uploads"), "upload directory")
	fs.StringVar(&c.DefaultAdminUser, "admin-user", env("ADMIN_USER", "ADMIN"), "default admin username")
	fs.StringVar(&c.DefaultAdminPass, "admin-pass", env("ADMIN_PASS", "123456"), "default admin password")
	fs.BoolVar(&c.AutoMigrate, "migrate", true, "auto-create tables on start")
	fs.BoolVar(&c.Seed, "seed", false, "seed sample data on start")

	_ = fs.Parse(os.Args[1:])

	o := dbOptions{Host: *dbHost, Port: *dbPort, User: *dbUser, Password: *dbPass, Database: *dbName}
	c.DSN = o.User + ":" + o.Password + "@tcp(" + o.Host + ":" + strconv.Itoa(o.Port) + ")/" + o.Database +
		"?charset=utf8mb4&parseTime=True&loc=Local&multiStatements=true"
	return c
}

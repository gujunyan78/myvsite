package model

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"myvsite/internal/config"
)

// DB 全局数据库连接
var DB *sql.DB

// Init 建立数据库连接，按需建表并写入默认管理员。
func Init(c *config.Config) error {
	db, err := sql.Open("mysql", c.DSN)
	if err != nil {
		return err
	}
	db.SetMaxOpenConns(30)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Hour)

	if err := db.Ping(); err != nil {
		return fmt.Errorf("Failed to connect to database: %w", err)
	}
	DB = db

	if c.AutoMigrate {
		if err := migrate(); err != nil {
			return err
		}
		if err := ensureAdmin(c.DefaultAdminUser, c.DefaultAdminPass); err != nil {
			return err
		}
	}
	if c.Seed {
		return seedData()
	}
	return nil
}

func migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS admins (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			username VARCHAR(64) NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			must_change_password TINYINT(1) NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY uk_admins_username (username)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

		`CREATE TABLE IF NOT EXISTS sessions (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			token VARCHAR(128) NOT NULL,
			admin_id BIGINT UNSIGNED NOT NULL,
			expires_at DATETIME NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY uk_sessions_token (token),
			KEY idx_sessions_admin (admin_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

		`CREATE TABLE IF NOT EXISTS bloggers (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			account VARCHAR(64) NOT NULL,
			description VARCHAR(500) NOT NULL DEFAULT '',
			avatar VARCHAR(500) NOT NULL DEFAULT '',
			status TINYINT(1) NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY uk_bloggers_account (account)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

		`CREATE TABLE IF NOT EXISTS videos (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			blogger_id BIGINT UNSIGNED NOT NULL,
			title VARCHAR(200) NOT NULL,
			description VARCHAR(1000) NOT NULL DEFAULT '',
			cover_url VARCHAR(1000) NOT NULL DEFAULT '',
			video_url VARCHAR(1000) NOT NULL,
			duration VARCHAR(32) NOT NULL DEFAULT '',
			width INT NOT NULL DEFAULT 0,
			height INT NOT NULL DEFAULT 0,
			likes BIGINT NOT NULL DEFAULT 0,
			comments BIGINT NOT NULL DEFAULT 0,
			views BIGINT NOT NULL DEFAULT 0,
			status TINYINT(1) NOT NULL DEFAULT 1,
			rand_key DOUBLE NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			KEY idx_videos_blogger (blogger_id),
			KEY idx_videos_created (created_at),
			KEY idx_videos_status (status),
			KEY idx_videos_rand (rand_key)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

		`CREATE TABLE IF NOT EXISTS comments (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			video_id BIGINT UNSIGNED NOT NULL,
			nickname VARCHAR(64) NOT NULL DEFAULT 'Guest',
			content TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			KEY idx_comments_video (video_id, id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

		`CREATE TABLE IF NOT EXISTS video_likes (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			video_id BIGINT UNSIGNED NOT NULL,
			client_id VARCHAR(64) NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY uk_likes (video_id, client_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	}
	for _, s := range stmts {
		if _, err := DB.Exec(s); err != nil {
			return fmt.Errorf("Failed to create tables: %w", err)
		}
	}
	return nil
}

func ensureAdmin(username, password string) error {
	var count int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM admins`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	_, err = DB.Exec(`INSERT INTO admins (username, password_hash, must_change_password) VALUES (?,?,1)`, username, string(hash))
	if err != nil {
		return err
	}
	log.Printf("Initialized admin account: %s (password change required on first login)", username)
	return nil
}

func seedData() error {
	var count int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM videos`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	type seedBlogger struct {
		account string
		desc    string
		avatar  string
	}
	bloggers := []seedBlogger{
		{"Travel Diaries", "30+ countries and counting — landscapes and stories from the road", "https://api.dicebear.com/7.x/avataaars/svg?seed=travel"},
		{"Tiny Kitchen", "One easy home-style recipe every day", "https://api.dicebear.com/7.x/avataaars/svg?seed=kitchen"},
		{"Geek Scope", "Honest tech and gadget reviews", "https://api.dicebear.com/7.x/avataaars/svg?seed=geek"},
		{"Cat Planet", "Daily life of two orange cats — pure healing", "https://api.dicebear.com/7.x/avataaars/svg?seed=cat"},
		{"Fit With Zhang", "Science-based training and nutrition — get stronger together", "https://api.dicebear.com/7.x/avataaars/svg?seed=fit"},
	}

	clips := []struct {
		title string
		cover string
		url   string
	}{
		{"First dance under the Iceland aurora", "/static/uploads/demo/demo1.jpg", "/static/uploads/demo/demo1.mp4"},
		{"10-minute tomato beef rice", "/static/uploads/demo/demo2.jpg", "/static/uploads/demo/demo2.mp4"},
		{"The best compact cameras of 2026", "/static/uploads/demo/demo3.jpg", "/static/uploads/demo/demo3.mp4"},
		{"Orange cats see snow for the first time", "/static/uploads/demo/demo4.jpg", "/static/uploads/demo/demo4.mp4"},
		{"Three moves to build core strength", "/static/uploads/demo/demo5.jpg", "/static/uploads/demo/demo5.mp4"},
		{"Late night at a Tokyo convenience store", "/static/uploads/demo/demo1.jpg", "/static/uploads/demo/demo1.mp4"},
		{"Juicy cheese-baked rice in one bite", "/static/uploads/demo/demo2.jpg", "/static/uploads/demo/demo2.mp4"},
		{"Teardown: is this laptop worth it?", "/static/uploads/demo/demo3.jpg", "/static/uploads/demo/demo3.mp4"},
		{"Cat scratcher buying guide", "/static/uploads/demo/demo4.jpg", "/static/uploads/demo/demo4.mp4"},
		{"What to eat on a muscle-gain diet", "/static/uploads/demo/demo5.jpg", "/static/uploads/demo/demo5.mp4"},
		{"Iceland road trip — day 3", "/static/uploads/demo/demo1.jpg", "/static/uploads/demo/demo1.mp4"},
		{"Pour-over coffee for beginners", "/static/uploads/demo/demo2.jpg", "/static/uploads/demo/demo2.mp4"},
		{"My full camping gear checklist", "/static/uploads/demo/demo3.jpg", "/static/uploads/demo/demo3.mp4"},
		{"Late-night coding desk makeover", "/static/uploads/demo/demo4.jpg", "/static/uploads/demo/demo4.mp4"},
		{"Seaside sunrise timelapse", "/static/uploads/demo/demo5.jpg", "/static/uploads/demo/demo5.mp4"},
		{"Low-calorie desserts anyone can make", "/static/uploads/demo/demo1.jpg", "/static/uploads/demo/demo1.mp4"},
		{"5 tricks for cinematic phone video", "/static/uploads/demo/demo2.jpg", "/static/uploads/demo/demo2.mp4"},
		{"How to actually choose cat food", "/static/uploads/demo/demo3.jpg", "/static/uploads/demo/demo3.mp4"},
		{"Key running form to avoid injury", "/static/uploads/demo/demo4.jpg", "/static/uploads/demo/demo4.mp4"},
		{"Hidden shops in the old town", "/static/uploads/demo/demo5.jpg", "/static/uploads/demo/demo5.mp4"},
	}

	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	bloggerIDs := make([]int64, 0, len(bloggers))
	for _, b := range bloggers {
		res, err := tx.Exec(`INSERT INTO bloggers (account, description, avatar) VALUES (?,?,?)`, b.account, b.desc, b.avatar)
		if err != nil {
			return err
		}
		id, _ := res.LastInsertId()
		bloggerIDs = append(bloggerIDs, id)
	}

	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i, v := range clips {
		bid := bloggerIDs[i%len(bloggerIDs)]
		likes := int64(rnd.Intn(5000))
		views := likes*20 + int64(rnd.Intn(10000))
		comments := int64(rnd.Intn(300))
		rk := rnd.Float64()
		if _, err := tx.Exec(`INSERT INTO videos
			(blogger_id, title, description, cover_url, video_url, likes, views, comments, rand_key, created_at)
			VALUES (?,?,?,?,?,?,?,?,?,DATE_SUB(NOW(), INTERVAL ? HOUR))`,
			bid, v.title, "This video is all about "+v.title+" — watch the full process!", v.cover, v.url, likes, views, comments, rk, i*7); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ErrNotFound 通用未找到错误
var ErrNotFound = errors.New("record not found")

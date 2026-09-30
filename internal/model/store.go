package model

import (
	"database/sql"
	"math/rand"
	"strings"
	"time"
)

// ---------- 数据结构 ----------

// Blogger 博主
type Blogger struct {
	ID          int64  `json:"id"`
	Account     string `json:"account"`
	Description string `json:"description"`
	Avatar      string `json:"avatar"`
	Status      int    `json:"status"`
	VideoCount  int64  `json:"video_count"`
	CreatedAt   string `json:"created_at"`
}

// Video 视频
type Video struct {
	ID          int64      `json:"id"`
	BloggerID   int64      `json:"blogger_id"`
	Blogger     *Blogger   `json:"blogger,omitempty"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	CoverURL    string     `json:"cover_url"`
	VideoURL    string     `json:"video_url"`
	Duration    string     `json:"duration"`
	Width       int        `json:"width"`
	Height      int        `json:"height"`
	Likes       int64      `json:"likes"`
	Comments    int64      `json:"comments"`
	Views       int64      `json:"views"`
	Status      int        `json:"status"`
	RandKey     float64    `json:"-"`
	Liked       bool       `json:"liked"`
	CreatedAt   string     `json:"created_at"`
	LatestTips  []*Comment `json:"latest_comments,omitempty"`
}

// Comment 评论
type Comment struct {
	ID        int64  `json:"id"`
	VideoID   int64  `json:"video_id"`
	Nickname  string `json:"nickname"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

// Admin 管理员
type Admin struct {
	ID                 int64  `json:"id"`
	Username           string `json:"username"`
	MustChangePassword bool   `json:"must_change_password"`
	CreatedAt          string `json:"created_at"`
}

// Paged 通用分页返回
type Paged struct {
	Items      interface{} `json:"items"`
	NextCursor string      `json:"next_cursor"`
	HasMore    bool        `json:"has_more"`
	Total      int64       `json:"total"`
}

func scanTime(v interface{}) string {
	switch t := v.(type) {
	case time.Time:
		return t.Format("2006-01-02 15:04:05")
	case sql.NullTime:
		if t.Valid {
			return t.Time.Format("2006-01-02 15:04:05")
		}
	}
	return ""
}

func nullString(v sql.NullString) string {
	if v.Valid {
		return v.String
	}
	return ""
}

// ---------- 博主 ----------

// CreateBlogger 新增博主
func CreateBlogger(account, description, avatar string) (*Blogger, error) {
	res, err := DB.Exec(`INSERT INTO bloggers (account, description, avatar) VALUES (?,?,?)`, account, description, avatar)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return GetBlogger(id)
}

// UpdateBlogger 更新博主
func UpdateBlogger(id int64, account, description, avatar string, status int) error {
	res, err := DB.Exec(`UPDATE bloggers SET account=?, description=?, avatar=?, status=? WHERE id=?`, account, description, avatar, status, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteBlogger 删除博主及其视频
func DeleteBlogger(id int64) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM comments WHERE video_id IN (SELECT id FROM videos WHERE blogger_id=?)`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM video_likes WHERE video_id IN (SELECT id FROM videos WHERE blogger_id=?)`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM videos WHERE blogger_id=?`, id); err != nil {
		return err
	}
	res, err := tx.Exec(`DELETE FROM bloggers WHERE id=?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return tx.Commit()
}

// GetBlogger 查询单个博主
func GetBlogger(id int64) (*Blogger, error) {
	row := DB.QueryRow(`SELECT id, account, description, avatar, status, created_at,
		(SELECT COUNT(*) FROM videos v WHERE v.blogger_id=bloggers.id AND v.status=1) AS video_count
		FROM bloggers WHERE id=?`, id)
	b, err := scanBlogger(row)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return b, err
}

// ListBloggers 后台博主列表
func ListBloggers(keyword string, page, pageSize int) ([]*Blogger, int64, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	if keyword != "" {
		where += " AND account LIKE ?"
		args = append(args, "%"+keyword+"%")
	}
	var total int64
	if err := DB.QueryRow("SELECT COUNT(*) FROM bloggers "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	q := `SELECT id, account, description, avatar, status, created_at,
		(SELECT COUNT(*) FROM videos v WHERE v.blogger_id=bloggers.id AND v.status=1) AS video_count
		FROM bloggers ` + where + ` ORDER BY id DESC LIMIT ? OFFSET ?`
	args = append(args, pageSize, offset)
	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	list := []*Blogger{}
	for rows.Next() {
		b, err := scanBlogger(rows)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, b)
	}
	return list, total, rows.Err()
}

type scannable interface {
	Scan(dest ...interface{}) error
}

func scanBlogger(row scannable) (*Blogger, error) {
	var (
		b      Blogger
		timeAt interface{}
	)
	err := row.Scan(&b.ID, &b.Account, &b.Description, &b.Avatar, &b.Status, &timeAt, &b.VideoCount)
	if err != nil {
		return nil, err
	}
	b.CreatedAt = scanTime(timeAt)
	return &b, nil
}

// ListPublicBloggers 前台博主列表
func ListPublicBloggers(limit int) ([]*Blogger, error) {
	rows, err := DB.Query(`SELECT id, account, description, avatar, status, created_at,
		(SELECT COUNT(*) FROM videos v WHERE v.blogger_id=bloggers.id AND v.status=1) AS video_count
		FROM bloggers WHERE status=1 ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []*Blogger{}
	for rows.Next() {
		b, err := scanBlogger(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, b)
	}
	return list, rows.Err()
}

// ---------- 视频 ----------

// status01 将状态归一化为 0/1，避免写入非预期值。
func status01(v int) int {
	if v == 0 {
		return 0
	}
	return 1
}

const videoSelectCols = `v.id, v.blogger_id, v.title, v.description, v.cover_url, v.video_url, v.duration,
	v.width, v.height, v.likes, v.comments, v.views, v.status, v.rand_key, v.created_at,
	b.id, b.account, b.description, b.avatar, b.status`

func scanVideo(rows *sql.Rows) (*Video, error) {
	var (
		v      Video
		b      Blogger
		bID    sql.NullInt64
		bIDesc sql.NullString
		bAv    sql.NullString
		bSt    sql.NullInt64
		cat    interface{}
	)
	err := rows.Scan(&v.ID, &v.BloggerID, &v.Title, &v.Description, &v.CoverURL, &v.VideoURL,
		&v.Duration, &v.Width, &v.Height, &v.Likes, &v.Comments, &v.Views, &v.Status, &v.RandKey, &cat,
		&bID, &b.Account, &bIDesc, &bAv, &bSt)
	if err != nil {
		return nil, err
	}
	v.CreatedAt = scanTime(cat)
	if bID.Valid {
		b.ID = bID.Int64
		b.Description = nullString(bIDesc)
		b.Avatar = nullString(bAv)
		if bSt.Valid {
			b.Status = int(bSt.Int64)
		}
		v.Blogger = &b
	}
	return &v, nil
}

// VideoFilter 视频列表过滤条件
type VideoFilter struct {
	BloggerID int64
	Keyword   string
	Page      int
	PageSize  int
	Random    bool
	Cursor    float64
	CursorSet bool
}

// ListVideos 前台视频列表：支持随机游标翻页（首页随机展示 + 无限下拉不重复）
func ListVideos(f VideoFilter) ([]*Video, bool, error) {
	limit := f.PageSize
	if limit <= 0 || limit > 60 {
		limit = 12
	}

	where := "WHERE v.status=1 AND b.status=1"
	args := []interface{}{}

	if f.BloggerID > 0 {
		where += " AND v.blogger_id=?"
		args = append(args, f.BloggerID)
	}
	if f.Keyword != "" {
		where += " AND (v.title LIKE ? OR v.description LIKE ? OR b.account LIKE ?)"
		kw := "%" + f.Keyword + "%"
		args = append(args, kw, kw, kw)
	}

	if !f.Random {
		offset := (f.Page - 1) * limit
		orderBy := "ORDER BY v.id DESC"
		q := `SELECT ` + videoSelectCols + ` FROM videos v JOIN bloggers b ON b.id=v.blogger_id ` + where + " " + orderBy + " LIMIT ? OFFSET ?"
		rows, err := DB.Query(q, append(args, limit+1, offset)...)
		if err != nil {
			return nil, false, err
		}
		defer rows.Close()
		list, err := collectVideos(rows)
		if err != nil {
			return nil, false, err
		}
		hasMore := len(list) > limit
		if hasMore {
			list = list[:limit]
		}
		return list, hasMore, nil
	}

	// 随机瀑布流：首屏随机起点，不足则回卷拼接，保证首屏有内容且下拉连续不重复
	return randomFeed(where, args, limit, f.CursorSet, f.Cursor)
}

// randomFeed 随机顺序游标翻页：以 rand_key 排序循环返回，首屏取随机起点
func randomFeed(baseWhere string, baseArgs []interface{}, limit int, cursorSet bool, cursor float64) ([]*Video, bool, error) {
	collect := func(pred string, val float64) ([]*Video, error) {
		q := `SELECT ` + videoSelectCols + ` FROM videos v JOIN bloggers b ON b.id=v.blogger_id ` + baseWhere + pred + " ORDER BY v.rand_key ASC LIMIT ?"
		rows, err := DB.Query(q, append(append([]interface{}{}, baseArgs...), val, limit+1)...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return collectVideos(rows)
	}
	var list []*Video
	if !cursorSet {
		r := rand.New(rand.NewSource(time.Now().UnixNano())).Float64()
		a, err := collect(" AND v.rand_key >= ?", r)
		if err != nil {
			return nil, false, err
		}
		list = append(list, a...)
		if len(list) < limit {
			b, _ := collect(" AND v.rand_key < ?", r)
			list = append(list, b...)
		}
	} else {
		a, err := collect(" AND v.rand_key > ?", cursor)
		if err != nil {
			return nil, false, err
		}
		list = append(list, a...)
		if len(list) < limit {
			b, _ := collect(" AND v.rand_key >= ?", 0.0)
			list = append(list, b...)
		}
	}
	hasMore := len(list) > limit
	if hasMore {
		list = list[:limit]
	}
	return list, hasMore, nil
}

// collectVideos 读取视频行集合
func collectVideos(rows *sql.Rows) ([]*Video, error) {
	list := []*Video{}
	for rows.Next() {
		v, err := scanVideo(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, v)
	}
	return list, rows.Err()
}

// CountVideos 统计有效视频数（过滤条件需与 ListVideos 保持一致）
func CountVideos(bloggerID int64, keyword string) (int64, error) {
	q := `SELECT COUNT(*) FROM videos v JOIN bloggers b ON b.id=v.blogger_id WHERE v.status=1 AND b.status=1`
	args := []interface{}{}
	if bloggerID > 0 {
		q += " AND v.blogger_id=?"
		args = append(args, bloggerID)
	}
	if keyword != "" {
		q += " AND (v.title LIKE ? OR v.description LIKE ? OR b.account LIKE ?)"
		kw := "%" + keyword + "%"
		args = append(args, kw, kw, kw)
	}
	var n int64
	err := DB.QueryRow(q, args...).Scan(&n)
	return n, err
}

// GetVideo 查询单个视频
func GetVideo(id int64) (*Video, error) {
	rows, err := DB.Query(`SELECT `+videoSelectCols+` FROM videos v JOIN bloggers b ON b.id=v.blogger_id WHERE v.id=?`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if rows.Next() {
		v, err := scanVideo(rows)
		if err != nil {
			return nil, err
		}
		return v, nil
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return nil, ErrNotFound
}

// CreateVideo 新增视频
func CreateVideo(bloggerID int64, title, description, coverURL, videoURL, duration string, width, height, status int) (*Video, error) {
	rk := rand.Float64()
	res, err := DB.Exec(`INSERT INTO videos
		(blogger_id, title, description, cover_url, video_url, duration, width, height, status, rand_key)
		VALUES (?,?,?,?,?,?,?,?,?,?)`, bloggerID, title, description, coverURL, videoURL, duration, width, height, status01(status), rk)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return GetVideo(id)
}

// UpdateVideo 更新视频
func UpdateVideo(id, bloggerID int64, title, description, coverURL, videoURL, duration string, width, height, status int) error {
	res, err := DB.Exec(`UPDATE videos SET blogger_id=?, title=?, description=?, cover_url=?, video_url=?,
		duration=?, width=?, height=?, status=? WHERE id=?`,
		bloggerID, title, description, coverURL, videoURL, duration, width, height, status01(status), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteVideo 删除视频
func DeleteVideo(id int64) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM comments WHERE video_id=?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM video_likes WHERE video_id=?`, id); err != nil {
		return err
	}
	res, err := tx.Exec(`DELETE FROM videos WHERE id=?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return tx.Commit()
}

// IncrViews 增加播放次数
func IncrViews(id int64) error {
	_, err := DB.Exec(`UPDATE videos SET views=views+1 WHERE id=?`, id)
	return err
}

// AdminListVideos 后台视频列表（含下线内容）
func AdminListVideos(keyword string, bloggerID int64, page, pageSize int) ([]*Video, int64, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	if bloggerID > 0 {
		where += " AND v.blogger_id=?"
		args = append(args, bloggerID)
	}
	if keyword != "" {
		where += " AND (v.title LIKE ? OR b.account LIKE ?)"
		args = append(args, "%"+keyword+"%", "%"+keyword+"%")
	}
	var total int64
	if err := DB.QueryRow(`SELECT COUNT(*) FROM videos v JOIN bloggers b ON b.id=v.blogger_id `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	rows, err := DB.Query(`SELECT `+videoSelectCols+` FROM videos v JOIN bloggers b ON b.id=v.blogger_id `+
		where+` ORDER BY v.id DESC LIMIT ? OFFSET ?`, append(args, pageSize, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	list := []*Video{}
	for rows.Next() {
		v, err := scanVideo(rows)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, v)
	}
	return list, total, rows.Err()
}

// ---------- 点赞 ----------

// ToggleLike 点赞/取消点赞，返回最新状态
func ToggleLike(videoID int64, clientID string) (liked bool, likes int64, err error) {
	var exists int
	if err = DB.QueryRow(`SELECT COUNT(*) FROM video_likes WHERE video_id=? AND client_id=?`, videoID, clientID).Scan(&exists); err != nil {
		return
	}
	tx, err := DB.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()
	if exists > 0 {
		if _, err = tx.Exec(`DELETE FROM video_likes WHERE video_id=? AND client_id=?`, videoID, clientID); err != nil {
			return
		}
		if _, err = tx.Exec(`UPDATE videos SET likes=GREATEST(likes-1,0) WHERE id=?`, videoID); err != nil {
			return
		}
		liked = false
	} else {
		if _, err = tx.Exec(`INSERT INTO video_likes (video_id, client_id) VALUES (?,?)`, videoID, clientID); err != nil {
			return
		}
		if _, err = tx.Exec(`UPDATE videos SET likes=likes+1 WHERE id=?`, videoID); err != nil {
			return
		}
		liked = true
	}
	if err = tx.QueryRow(`SELECT likes FROM videos WHERE id=?`, videoID).Scan(&likes); err != nil {
		return
	}
	err = tx.Commit()
	return
}

// MyLikes 查询某客户端在指定视频上的点赞情况
func MyLikes(clientID string, videoIDs []int64) (map[int64]bool, error) {
	m := map[int64]bool{}
	if len(videoIDs) == 0 || clientID == "" {
		return m, nil
	}
	placeholders := strings.Repeat("?,", len(videoIDs))
	placeholders = placeholders[:len(placeholders)-1]
	args := make([]interface{}, 0, len(videoIDs)+1)
	args = append(args, clientID)
	for _, id := range videoIDs {
		args = append(args, id)
	}
	rows, err := DB.Query(`SELECT video_id FROM video_likes WHERE client_id=? AND video_id IN (`+placeholders+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		m[id] = true
	}
	return m, rows.Err()
}

// ---------- 评论 ----------

// ListComments 评论列表（倒序）
func ListComments(videoID int64, page, pageSize int) ([]*Comment, int64, error) {
	var total int64
	if err := DB.QueryRow(`SELECT COUNT(*) FROM comments WHERE video_id=?`, videoID).Scan(&total); err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	rows, err := DB.Query(`SELECT id, video_id, nickname, content, created_at FROM comments WHERE video_id=? ORDER BY id DESC LIMIT ? OFFSET ?`,
		videoID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	list := []*Comment{}
	for rows.Next() {
		var c Comment
		var cat interface{}
		if err := rows.Scan(&c.ID, &c.VideoID, &c.Nickname, &c.Content, &cat); err != nil {
			return nil, 0, err
		}
		c.CreatedAt = scanTime(cat)
		list = append(list, &c)
	}
	return list, total, rows.Err()
}

// CreateComment 新增评论
func CreateComment(videoID int64, nickname, content string) (*Comment, error) {
	tx, err := DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`INSERT INTO comments (video_id, nickname, content) VALUES (?,?,?)`, videoID, nickname, content)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	if _, err := tx.Exec(`UPDATE videos SET comments=comments+1 WHERE id=?`, videoID); err != nil {
		return nil, err
	}
	var c Comment
	var cat interface{}
	if err := tx.QueryRow(`SELECT id, video_id, nickname, content, created_at FROM comments WHERE id=?`, id).
		Scan(&c.ID, &c.VideoID, &c.Nickname, &c.Content, &cat); err != nil {
		return nil, err
	}
	c.CreatedAt = scanTime(cat)
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &c, nil
}

// DeleteComment 删除评论
func DeleteComment(id int64) error {
	var videoID int64
	if err := DB.QueryRow(`SELECT video_id FROM comments WHERE id=?`, id).Scan(&videoID); err != nil {
		if err == sql.ErrNoRows {
			return ErrNotFound
		}
		return err
	}
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM comments WHERE id=?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE videos SET comments=GREATEST(comments-1,0) WHERE id=?`, videoID); err != nil {
		return err
	}
	return tx.Commit()
}

// ---------- 管理员与会话 ----------

// AuthAdmin 校验账号密码
func AuthAdmin(username, password string) (*Admin, error) {
	var (
		a    Admin
		hash string
		cat  interface{}
	)
	err := DB.QueryRow(`SELECT id, username, password_hash, must_change_password, created_at FROM admins WHERE username=?`, username).
		Scan(&a.ID, &a.Username, &hash, &a.MustChangePassword, &cat)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	a.CreatedAt = scanTime(cat)
	if err := compareHash(hash, password); err != nil {
		return nil, ErrNotFound
	}
	return &a, nil
}

// GetAdmin 查询管理员
func GetAdmin(id int64) (*Admin, error) {
	var (
		a   Admin
		cat interface{}
	)
	err := DB.QueryRow(`SELECT id, username, must_change_password, created_at FROM admins WHERE id=?`, id).
		Scan(&a.ID, &a.Username, &a.MustChangePassword, &cat)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	a.CreatedAt = scanTime(cat)
	return &a, nil
}

// UpdateAdminPasswordAndUsername 修改密码（可同时修改账号）
func UpdateAdminPasswordAndUsername(id int64, newUser, newPass string) error {
	hash, err := hashPassword(newPass)
	if err != nil {
		return err
	}
	res, err := DB.Exec(`UPDATE admins SET username=?, password_hash=?, must_change_password=0 WHERE id=?`, newUser, hash, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// CreateSession 创建会话
func CreateSession(adminID int64, token string, ttl time.Duration) error {
	_, err := DB.Exec(`INSERT INTO sessions (token, admin_id, expires_at) VALUES (?,?, DATE_ADD(NOW(), INTERVAL ? SECOND))`,
		token, adminID, int64(ttl.Seconds()))
	return err
}

// GetSessionAdmin 通过 token 获取管理员
func GetSessionAdmin(token string) (*Admin, error) {
	var (
		a   Admin
		cat interface{}
	)
	err := DB.QueryRow(`SELECT a.id, a.username, a.must_change_password, a.created_at
		FROM sessions s JOIN admins a ON a.id=s.admin_id
		WHERE s.token=? AND s.expires_at > NOW()`, token).
		Scan(&a.ID, &a.Username, &a.MustChangePassword, &cat)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	a.CreatedAt = scanTime(cat)
	return &a, nil
}

// DeleteSession 删除会话
func DeleteSession(token string) error {
	_, err := DB.Exec(`DELETE FROM sessions WHERE token=?`, token)
	return err
}

// CleanExpiredSessions 清理过期会话
func CleanExpiredSessions() {
	_, _ = DB.Exec(`DELETE FROM sessions WHERE expires_at < NOW()`)
}

// Stats 后台首页统计
func Stats() (map[string]int64, error) {
	queries := []string{
		`SELECT COUNT(*) FROM videos WHERE status=1`,
		`SELECT COUNT(*) FROM bloggers WHERE status=1`,
		`SELECT IFNULL(SUM(views),0) FROM videos`,
		`SELECT IFNULL(SUM(likes),0) FROM videos`,
		`SELECT COUNT(*) FROM comments`,
	}
	keys := []string{"videos", "bloggers", "views", "likes", "comments"}
	out := map[string]int64{}
	for i, q := range queries {
		var v int64
		if err := DB.QueryRow(q).Scan(&v); err != nil {
			return nil, err
		}
		out[keys[i]] = v
	}
	return out, nil
}

package handler

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"myvsite/internal/model"
)

const sessionTTL = 12 * time.Hour

var uploadDir string

// SetUploadDir 设置上传目录
func SetUploadDir(dir string) error {
	uploadDir = dir
	return os.MkdirAll(dir, 0o755)
}

func newToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Login POST /api/admin/login
func Login(c *gin.Context) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, "Invalid request payload")
		return
	}
	a, err := model.AuthAdmin(strings.TrimSpace(body.Username), body.Password)
	if err == model.ErrNotFound {
		fail(c, 401, http.StatusUnauthorized, "Incorrect username or password")
		return
	}
	if err != nil {
		serverErr(c, "Login failed: "+err.Error())
		return
	}
	token := newToken()
	if err := model.CreateSession(a.ID, token, sessionTTL); err != nil {
		serverErr(c, "Failed to create session: "+err.Error())
		return
	}
	model.CleanExpiredSessions()
	ok(c, gin.H{
		"token":                token,
		"expires_in":           int(sessionTTL.Seconds()),
		"username":             a.Username,
		"must_change_password": a.MustChangePassword,
	})
}

// Logout POST /api/admin/logout
func Logout(c *gin.Context) {
	if token := requestToken(c); token != "" {
		_ = model.DeleteSession(token)
	}
	ok(c, gin.H{"logout": true})
}

// requestToken 从 Header / Authorization / query 中解析管理员 token
func requestToken(c *gin.Context) string {
	if token := strings.TrimSpace(c.GetHeader("X-Admin-Token")); token != "" {
		return token
	}
	if auth := c.GetHeader("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	}
	return strings.TrimSpace(c.Query("token"))
}

// Profile GET /api/admin/profile
func Profile(c *gin.Context) {
	a, exists := c.Get("admin")
	if !exists {
		unauthorized(c, "Not signed in")
		return
	}
	ok(c, a)
}

// ChangePassword POST /api/admin/password
func ChangePassword(c *gin.Context) {
	admin, okAdmin := c.Get("admin")
	if !okAdmin {
		unauthorized(c, "Not signed in")
		return
	}
	a := admin.(*model.Admin)

	var body struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
		Username    string `json:"username"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, "Invalid request payload")
		return
	}
	newPass := strings.TrimSpace(body.NewPassword)
	if len(newPass) < 6 {
		badRequest(c, "New password must be at least 6 characters")
		return
	}
	if newPass == body.OldPassword {
		badRequest(c, "New password must differ from the old one")
		return
	}

	// 非强制修改场景需要校验原密码
	if !a.MustChangePassword {
		if _, err := model.AuthAdmin(a.Username, body.OldPassword); err != nil {
			fail(c, 401, http.StatusUnauthorized, "Current password is incorrect")
			return
		}
	}

	newUser := strings.TrimSpace(body.Username)
	if newUser == "" {
		newUser = a.Username
	}
	if err := model.UpdateAdminPasswordAndUsername(a.ID, newUser, newPass); err != nil {
		serverErr(c, "Update failed: "+err.Error())
		return
	}
	// 使当前会话失效：token 可能来自 Header / Authorization / query，需逐一解析
	if token := requestToken(c); token != "" {
		_ = model.DeleteSession(token)
	}
	ok(c, gin.H{"username": newUser, "must_change_password": false})
}

// Stats GET /api/admin/stats
func Stats(c *gin.Context) {
	s, err := model.Stats()
	if err != nil {
		serverErr(c, err.Error())
		return
	}
	ok(c, s)
}

// AdminListBloggers GET /api/admin/bloggers
func AdminListBloggers(c *gin.Context) {
	page := int(toInt64(c.DefaultQuery("page", "1")))
	size := int(toInt64(c.DefaultQuery("page_size", "20")))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	list, total, err := model.ListBloggers(strings.TrimSpace(c.Query("keyword")), page, size)
	if err != nil {
		serverErr(c, err.Error())
		return
	}
	ok(c, gin.H{"items": list, "total": total, "page": page, "page_size": size})
}

// AdminCreateBlogger POST /api/admin/bloggers {account, description, avatar}
func AdminCreateBlogger(c *gin.Context) {
	var body struct {
		Account     string `json:"account"`
		Description string `json:"description"`
		Avatar      string `json:"avatar"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, "Invalid request payload")
		return
	}
	account := strings.TrimSpace(body.Account)
	if account == "" {
		badRequest(c, "Creator account is required")
		return
	}
	if len([]rune(account)) > 64 {
		badRequest(c, "Creator account must be 64 characters or fewer")
		return
	}
	b, err := model.CreateBlogger(account, strings.TrimSpace(body.Description), strings.TrimSpace(body.Avatar))
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate") {
			badRequest(c, "This creator account already exists")
			return
		}
		serverErr(c, "Create failed: "+err.Error())
		return
	}
	ok(c, b)
}

// AdminUpdateBlogger PUT /api/admin/bloggers/:id
func AdminUpdateBlogger(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		badRequest(c, "Invalid ID")
		return
	}
	var body struct {
		Account     string `json:"account"`
		Description string `json:"description"`
		Avatar      string `json:"avatar"`
		Status      int    `json:"status"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, "Invalid request payload")
		return
	}
	account := strings.TrimSpace(body.Account)
	if account == "" {
		badRequest(c, "Creator account is required")
		return
	}
	if err := model.UpdateBlogger(id, account, strings.TrimSpace(body.Description), strings.TrimSpace(body.Avatar), boolToInt(body.Status)); err != nil {
		if err == model.ErrNotFound {
			notFound(c, "Creator not found")
			return
		}
		serverErr(c, "Update failed: "+err.Error())
		return
	}
	b, err := model.GetBlogger(id)
	if err != nil {
		serverErr(c, err.Error())
		return
	}
	ok(c, b)
}

// AdminDeleteBlogger DELETE /api/admin/bloggers/:id
func AdminDeleteBlogger(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		badRequest(c, "Invalid ID")
		return
	}
	if err := model.DeleteBlogger(id); err != nil {
		if err == model.ErrNotFound {
			notFound(c, "Creator not found")
			return
		}
		serverErr(c, "Delete failed: "+err.Error())
		return
	}
	ok(c, gin.H{"id": id})
}

// AdminListVideos GET /api/admin/videos
func AdminListVideos(c *gin.Context) {
	page := int(toInt64(c.DefaultQuery("page", "1")))
	size := int(toInt64(c.DefaultQuery("page_size", "20")))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	list, total, err := model.AdminListVideos(strings.TrimSpace(c.Query("keyword")), toInt64(c.Query("blogger_id")), page, size)
	if err != nil {
		serverErr(c, err.Error())
		return
	}
	ok(c, gin.H{"items": list, "total": total, "page": page, "page_size": size})
}

type videoPayload struct {
	BloggerID   int64  `json:"blogger_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	CoverURL    string `json:"cover_url"`
	VideoURL    string `json:"video_url"`
	Duration    string `json:"duration"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	// Status 用指针区分「未传」（默认上线）与「显式传 0」（下线）
	Status *int `json:"status"`
}

func (p videoPayload) validate() error {
	if p.BloggerID <= 0 {
		return errors.New("A creator must be selected")
	}
	if strings.TrimSpace(p.Title) == "" {
		return errors.New("Video title is required")
	}
	if strings.TrimSpace(p.VideoURL) == "" {
		return errors.New("Video URL is required")
	}
	if !strings.HasPrefix(p.VideoURL, "http://") && !strings.HasPrefix(p.VideoURL, "https://") && !strings.HasPrefix(p.VideoURL, "/") {
		return errors.New("Video URL must start with http(s):// or /")
	}
	if p.CoverURL != "" && !strings.HasPrefix(p.CoverURL, "http://") && !strings.HasPrefix(p.CoverURL, "https://") && !strings.HasPrefix(p.CoverURL, "/") {
		return errors.New("Cover URL must start with http(s):// or /")
	}
	return nil
}

// AdminCreateVideo POST /api/admin/videos
func AdminCreateVideo(c *gin.Context) {
	var body videoPayload
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, "Invalid request payload")
		return
	}
	body.Title = strings.TrimSpace(body.Title)
	body.Description = strings.TrimSpace(body.Description)
	body.CoverURL = strings.TrimSpace(body.CoverURL)
	body.VideoURL = strings.TrimSpace(body.VideoURL)
	status := 1
	if body.Status != nil {
		status = boolToInt(*body.Status)
	}
	if err := body.validate(); err != nil {
		badRequest(c, err.Error())
		return
	}
	if _, err := model.GetBlogger(body.BloggerID); err == model.ErrNotFound {
		badRequest(c, "Selected creator does not exist")
		return
	}
	v, err := model.CreateVideo(body.BloggerID, body.Title, body.Description, body.CoverURL, body.VideoURL,
		strings.TrimSpace(body.Duration), body.Width, body.Height, status)
	if err != nil {
		serverErr(c, "Create failed: "+err.Error())
		return
	}
	ok(c, v)
}

// AdminUpdateVideo PUT /api/admin/videos/:id
func AdminUpdateVideo(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		badRequest(c, "Invalid ID")
		return
	}
	var body videoPayload
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, "Invalid request payload")
		return
	}
	body.Title = strings.TrimSpace(body.Title)
	body.Description = strings.TrimSpace(body.Description)
	body.CoverURL = strings.TrimSpace(body.CoverURL)
	body.VideoURL = strings.TrimSpace(body.VideoURL)
	if err := body.validate(); err != nil {
		badRequest(c, err.Error())
		return
	}
	status := 1
	if body.Status != nil {
		status = boolToInt(*body.Status)
	}
	if err := model.UpdateVideo(id, body.BloggerID, body.Title, body.Description, body.CoverURL, body.VideoURL,
		strings.TrimSpace(body.Duration), body.Width, body.Height, status); err != nil {
		if err == model.ErrNotFound {
			notFound(c, "Video not found")
			return
		}
		serverErr(c, "Update failed: "+err.Error())
		return
	}
	v, err := model.GetVideo(id)
	if err != nil {
		serverErr(c, err.Error())
		return
	}
	ok(c, v)
}

// AdminDeleteVideo DELETE /api/admin/videos/:id
func AdminDeleteVideo(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		badRequest(c, "Invalid ID")
		return
	}
	if err := model.DeleteVideo(id); err != nil {
		if err == model.ErrNotFound {
			notFound(c, "Video not found")
			return
		}
		serverErr(c, "Delete failed: "+err.Error())
		return
	}
	ok(c, gin.H{"id": id})
}

// AdminDeleteComment DELETE /api/admin/comments/:id
func AdminDeleteComment(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		badRequest(c, "Invalid ID")
		return
	}
	if err := model.DeleteComment(id); err != nil {
		if err == model.ErrNotFound {
			notFound(c, "Comment not found")
			return
		}
		serverErr(c, "Delete failed: "+err.Error())
		return
	}
	ok(c, gin.H{"id": id})
}

// Upload POST /api/admin/upload  字段名 file
func Upload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		badRequest(c, "Please choose a file to upload")
		return
	}
	if file.Size > 200*1024*1024 {
		badRequest(c, "File must not exceed 200MB")
		return
	}
	src, err := file.Open()
	if err != nil {
		serverErr(c, "Failed to read file")
		return
	}
	defer src.Close()

	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowed := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
		".mp4": true, ".webm": true, ".m3u8": true, ".mov": true}
	if ext == "" || !allowed[ext] {
		badRequest(c, "Unsupported file type")
		return
	}
	name := time.Now().Format("20060102150405") + "-" + strconv.FormatInt(time.Now().UnixNano()%1e6, 10) + ext
	dst := filepath.Join(uploadDir, name)
	out, err := os.Create(dst)
	if err != nil {
		serverErr(c, "Failed to save file")
		return
	}
	if _, err := io.Copy(out, src); err != nil {
		out.Close()
		serverErr(c, "Failed to write file")
		return
	}
	out.Close()

	ok(c, gin.H{"url": "/static/uploads/" + name, "name": name, "size": file.Size})
}

func boolToInt(v int) int {
	if v != 0 {
		return 1
	}
	return 0
}

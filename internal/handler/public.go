package handler

import (
	"math/rand"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"myvsite/internal/model"
)

type resp struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func ok(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, resp{Code: 0, Message: "ok", Data: data})
}

func fail(c *gin.Context, code int, status int, msg string) {
	c.JSON(status, resp{Code: code, Message: msg})
}

func badRequest(c *gin.Context, msg string)   { fail(c, 400, http.StatusBadRequest, msg) }
func unauthorized(c *gin.Context, msg string) { fail(c, 401, http.StatusUnauthorized, msg) }
func notFound(c *gin.Context, msg string)     { fail(c, 404, http.StatusNotFound, msg) }
func serverErr(c *gin.Context, msg string)    { fail(c, 500, http.StatusInternalServerError, msg) }

// ListVideos GET /api/videos
// query: blogger_id, cursor(随机游标), page, page_size, client_id
func ListVideos(c *gin.Context) {
	filter := model.VideoFilter{
		BloggerID: toInt64(c.Query("blogger_id")),
		Keyword:   strings.TrimSpace(c.Query("keyword")),
		Random:    c.Query("order") != "latest",
	}
	if !filter.Random {
		filter.Page = int(toInt64(c.DefaultQuery("page", "1")))
	}
	filter.PageSize = int(toInt64(c.DefaultQuery("page_size", "12")))
	if cur := c.Query("cursor"); cur != "" {
		if fv, err := strconv.ParseFloat(cur, 64); err == nil {
			filter.Cursor = fv
			filter.CursorSet = true
			filter.Random = true
		}
	} else if c.Query("random") == "1" {
		filter.Random = true
	}

	// 随机模式下首屏自动取随机起点，使"每一屏都不一样"
	if filter.Random && !filter.CursorSet && filter.BloggerID == 0 && filter.Keyword == "" {
		filter.Cursor = rand.Float64()
		filter.CursorSet = true
	}

	items, hasMore, err := model.ListVideos(filter)
	if err != nil {
		serverErr(c, "Failed to query videos: "+err.Error())
		return
	}

	// 随机模式下走到末尾时回卷到起点，保证下拉可以继续加载
	total, _ := model.CountVideos(filter.BloggerID, filter.Keyword)
	nextCursor := ""
	if filter.Random {
		if len(items) > 0 {
			nextCursor = strconv.FormatFloat(items[len(items)-1].RandKey, 'f', -1, 64)
		}
		if !hasMore {
			if total > 0 {
				nextCursor = "0"
				hasMore = true
			} else {
				hasMore = false
			}
		}
	}

	if clientID := strings.TrimSpace(c.Query("client_id")); clientID != "" && len(items) > 0 {
		ids := make([]int64, 0, len(items))
		for _, v := range items {
			ids = append(ids, v.ID)
		}
		if likedMap, err := model.MyLikes(clientID, ids); err == nil {
			for _, v := range items {
				v.Liked = likedMap[v.ID]
			}
		}
	}

	ok(c, gin.H{
		"items":       items,
		"has_more":    hasMore,
		"next_cursor": nextCursor,
		"total":       total,
	})
}

// GetVideo GET /api/videos/:id
func GetVideo(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		badRequest(c, "Invalid video ID")
		return
	}
	v, err := model.GetVideo(id)
	if err == model.ErrNotFound {
		notFound(c, "Video not found")
		return
	}
	if err != nil {
		serverErr(c, err.Error())
		return
	}
	if cid := c.Query("client_id"); cid != "" {
		if m, err := model.MyLikes(cid, []int64{v.ID}); err == nil {
			v.Liked = m[v.ID]
		}
	}
	comments, total, _ := model.ListComments(v.ID, 1, 5)
	if len(comments) > 0 {
		v.LatestTips = comments
		_ = total
	}
	ok(c, v)
}

// IncrViews POST /api/videos/:id/view
func IncrViews(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		badRequest(c, "Invalid video ID")
		return
	}
	if err := model.IncrViews(id); err != nil {
		serverErr(c, err.Error())
		return
	}
	ok(c, gin.H{"video_id": id})
}

// ToggleLike POST /api/videos/:id/like  {client_id}
func ToggleLike(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		badRequest(c, "Invalid video ID")
		return
	}
	var body struct {
		ClientID string `json:"client_id"`
	}
	_ = c.ShouldBindJSON(&body)
	clientID := strings.TrimSpace(body.ClientID)
	if clientID == "" {
		clientID = strings.TrimSpace(c.Query("client_id"))
	}
	if clientID == "" {
		badRequest(c, "Missing client_id")
		return
	}
	liked, likes, err := model.ToggleLike(id, clientID)
	if err != nil {
		serverErr(c, "Operation failed: "+err.Error())
		return
	}
	ok(c, gin.H{"video_id": id, "liked": liked, "likes": likes})
}

// ListComments GET /api/videos/:id/comments
func ListComments(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		badRequest(c, "Invalid video ID")
		return
	}
	page := int(toInt64(c.DefaultQuery("page", "1")))
	size := int(toInt64(c.DefaultQuery("page_size", "20")))
	list, total, err := model.ListComments(id, page, size)
	if err != nil {
		serverErr(c, err.Error())
		return
	}
	ok(c, gin.H{"items": list, "total": total, "page": page, "page_size": size})
}

// CreateComment POST /api/videos/:id/comments {nickname, content}
func CreateComment(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		badRequest(c, "Invalid video ID")
		return
	}
	var body struct {
		Nickname string `json:"nickname"`
		Content  string `json:"content"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, "Invalid request payload")
		return
	}
	content := strings.TrimSpace(body.Content)
	if content == "" {
		badRequest(c, "Comment content is required")
		return
	}
	if len([]rune(content)) > 200 {
		badRequest(c, "Comment must be 200 characters or fewer")
		return
	}
	nickname := strings.TrimSpace(body.Nickname)
	if nickname == "" {
		nickname = "Guest"
	}
	if len([]rune(nickname)) > 24 {
		nickname = string([]rune(nickname)[:24])
	}
	if _, err := model.GetVideo(id); err == model.ErrNotFound {
		notFound(c, "Video not found")
		return
	}
	comment, err := model.CreateComment(id, nickname, content)
	if err != nil {
		serverErr(c, "Failed to post comment: "+err.Error())
		return
	}
	ok(c, comment)
}

// ListBloggers GET /api/bloggers
func ListBloggers(c *gin.Context) {
	size := int(toInt64(c.DefaultQuery("page_size", "20")))
	if size <= 0 || size > 100 {
		size = 20
	}
	list, err := model.ListPublicBloggers(size)
	if err != nil {
		serverErr(c, err.Error())
		return
	}
	ok(c, gin.H{"items": list, "total": len(list)})
}

// GetBlogger GET /api/bloggers/:id
func GetBlogger(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		badRequest(c, "Invalid creator ID")
		return
	}
	b, err := model.GetBlogger(id)
	if err == model.ErrNotFound {
		notFound(c, "Creator not found")
		return
	}
	if err != nil {
		serverErr(c, err.Error())
		return
	}
	ok(c, b)
}

func toInt64(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return v
}

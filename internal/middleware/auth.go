package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"myvsite/internal/model"
)

const ctxAdminKey = "admin"

// AdminAuth 校验后台管理员 token；未修改初始密码时仅放行改密接口
func AdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := strings.TrimSpace(c.GetHeader("X-Admin-Token"))
		if token == "" {
			if auth := c.GetHeader("Authorization"); strings.HasPrefix(auth, "Bearer ") {
				token = strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
			}
		}
		if token == "" {
			token = strings.TrimSpace(c.Query("token"))
		}
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "Not signed in or session expired"})
			return
		}
		a, err := model.GetSessionAdmin(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "Not signed in or session expired"})
			return
		}
		c.Set(ctxAdminKey, a)

		// 首次登录未修改密码时，除改密/登出外全部拦截
		if a.MustChangePassword {
			p := c.FullPath()
			if !strings.HasSuffix(p, "/api/admin/password") &&
				!strings.HasSuffix(p, "/api/admin/change-password") &&
				!strings.HasSuffix(p, "/api/admin/logout") {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "message": "Please change your password before using the admin panel", "need_change_password": true})
				return
			}
		}
		c.Next()
	}
}

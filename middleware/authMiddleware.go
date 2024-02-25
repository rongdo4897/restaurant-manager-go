package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rongdo4897/restaurant-manager-go/helpers"
)

func Authentication() gin.HandlerFunc {
	return func(c *gin.Context) {
		// lấy giá trị của token từ header "token" của yêu cầu.
		clientToken := c.Request.Header.Get("token")
		if clientToken == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "No authorization header provided"})
			return
		}

		// validate token
		claims, msg := helpers.ValidateToken(clientToken)
		if claims == nil || msg != "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "An error occurred - " + msg})
			return
		}

		// Nếu token hợp lệ, các thông tin từ claims sẽ được trích xuất và đặt vào các context của Gin
		// bằng cách sử dụng c.Set(). Các thông tin này sau đó có thể được truy cập từ các xử lý yêu cầu
		// khác trong chuỗi middleware.
		c.Set("email", claims.Email)
		c.Set("first_name", claims.First_name)
		c.Set("last_name", claims.Last_name)
		c.Set("uid", claims.Uid)

		// c.Next() chuyển quyền điều khiển cho middleware tiếp theo trong chuỗi middleware của Gin.
		c.Next()
	}
}

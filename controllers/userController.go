package controllers

import "github.com/gin-gonic/gin"

func GetUsers() gin.HandlerFunc {
	return func(c *gin.Context) {

	}
}

func GetUser() gin.HandlerFunc {
	return func(c *gin.Context) {

	}
}

func SignUp() gin.HandlerFunc {
	return func(c *gin.Context) {

	}
}

func Login() gin.HandlerFunc {
	return func(c *gin.Context) {

	}
}

// Chuyển đổi mật khẩu đầu vào thành 1 chuỗi không thể bị đảo ngược
func HashPassword(password string) string {
	return ""
}

// So sánh xác thực mật khẩu
func VerifyPassword(userPassword, providePassword string) (string, error) {
	return "", nil
}

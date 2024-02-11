package controllers

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rongdo4897/restaurant-manager-go/database"
	"go.mongodb.org/mongo-driver/bson"
)

// Kết nối tới bảng `menu`
var menuCollection = database.OpenCollection(database.Client, "menu")

func GetMenus() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		// Find() được sử dụng để truy vấn tất cả các tài liệu trong bộ sưu tập.
		// Trong trường hợp này, context.TODO() được sử dụng để tạo một ngữ cảnh mặc định (context) không có thông tin bổ sung.
		// bson.M{} là một bộ lọc trống, chỉ đơn giản là yêu cầu tất cả các tài liệu.
		result, err := menuCollection.Find(context.TODO(), bson.M{})
		defer cancel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while listing menu items"})
			return
		}

		// bson.M là một kiểu dữ liệu trong Go được sử dụng để biểu diễn các tài liệu MongoDB dưới dạng các cặp khóa-giá trị (MAP).
		/*
						doc := bson.M{
			    			"name":  "John Doe",
			    			"age":   30,
			    			"email": "john@example.com",
						}
		*/
		var allMenus []bson.M
		// Gán lại tất cả từ mongo.Cursor `result` vào mảng `allMenus` để trả về chuỗi json có kết quả là MAP
		if err = result.All(ctx, &allMenus); err != nil {
			log.Fatal(err)
		}

		c.JSON(http.StatusOK, allMenus)
	}
}

func GetMenu() gin.HandlerFunc {
	return func(c *gin.Context) {

	}
}

func CreateMenu() gin.HandlerFunc {
	return func(c *gin.Context) {

	}
}

func UpdateMenu() gin.HandlerFunc {
	return func(c *gin.Context) {

	}
}

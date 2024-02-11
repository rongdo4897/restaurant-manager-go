package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/rongdo4897/restaurant-manager-go/database"
	"github.com/rongdo4897/restaurant-manager-go/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Kết nối tới bảng `food`
var foodCollection = database.OpenCollection(database.Client, "food")

// Tạo 1 đối tượng Validate
var validate = validator.New()

func GetFoods() gin.HandlerFunc {
	return func(c *gin.Context) {

	}
}

func GetFood() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()
		foodId := c.Param("food_id")
		var foodModel models.Food

		// Trả về 1 đối tượng food từ `food_id` được chỉ định, đối tượng nhận về được tham chiếu lại vào `foodModel`
		err := foodCollection.FindOne(ctx, bson.M{"food_id": foodId}).Decode(&foodModel)
		// Trả về lỗi nếu tồn tại
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while fetching the food item"})
			return
		}

		c.JSON(http.StatusOK, foodModel)
	}
}

func CreateFood() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()
		var foodModel models.Food
		var menuModel models.Menu

		// Kiểm tra xem yêu cầu từ http có tham chiếu được tới `foodModel` không
		if err := c.BindJSON(&foodModel); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}

		// validate.Struct() được sử dụng để kiểm tra xem một biến cấu trúc có đáp ứng được các quy tắc kiểm tra hợp lệ (validation rules) đã được xác định hay không.
		// Quy tắc kiểm tra này thường được định nghĩa thông qua các tag validate trong các trường của cấu trúc.
		validationErr := validate.Struct(foodModel)
		if validationErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
			return
		}

		// Trả về 1 đối tượng menu từ `menu_id` được chỉ định, đối tượng nhận về được tham chiếu lại vào `menuModel`
		// Menu_id được chỉ định lấy từ http đã được tham chiếu vào `foodModel`
		err := menuCollection.FindOne(ctx, bson.M{"menu_id": foodModel.Menu_id}).Decode(&menuModel)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "menu was not found"})
			return
		}

		// Gán lại các giá trị khác
		foodModel.Created_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		foodModel.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		foodModel.Food_id = primitive.NewObjectID().Hex()
		var number = toFixed(*foodModel.Price, 2)
		foodModel.Price = &number

		// Insert foodModel vào bảng `food`
		result, insertErr := foodCollection.InsertOne(ctx, foodModel)
		if insertErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "food item was not created"})
			return
		}
		defer cancel()

		c.JSON(http.StatusOK, result)
	}
}

func UpdateFood() gin.HandlerFunc {
	return func(c *gin.Context) {

	}
}

func round(num float64) int {
	return 0
}

func toFixed(num float64, precision int) float64 {
	return 0
}

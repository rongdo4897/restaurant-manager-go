package controllers

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rongdo4897/restaurant-manager-go/database"
	"github.com/rongdo4897/restaurant-manager-go/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var orderCollection = database.OpenCollection(database.Client, "order")

func GetOrders() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		// Find() được sử dụng để truy vấn tất cả các tài liệu trong bộ sưu tập.
		// Trong trường hợp này, context.TODO() được sử dụng để tạo một ngữ cảnh mặc định (context) không có thông tin bổ sung.
		// bson.M{} là một bộ lọc trống, chỉ đơn giản là yêu cầu tất cả các tài liệu.
		result, err := orderCollection.Find(context.TODO(), bson.M{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while listing order items"})
			return
		}

		var allOrders []bson.M
		if err = result.All(ctx, &allOrders); err != nil {
			log.Fatal(err)
		}

		c.JSON(http.StatusOK, allOrders)
	}
}

func GetOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()
		orderId := c.Param("order_id")
		var orderModel models.Order

		// Trả về 1 đối tượng order từ `order_id` được chỉ định, đối tượng nhận về được tham chiếu lại vào `orderModel`
		err := orderCollection.FindOne(ctx, bson.M{"order_id": orderId}).Decode(&orderModel)
		// Trả về lỗi nếu tồn tại
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while fetching the order item"})
			return
		}

		c.JSON(http.StatusOK, orderModel)
	}
}

func CreateOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

		var orderModel models.Order
		var tableModel models.Table

		// Kiểm tra xem yêu cầu từ http có tham chiếu được tới `orderModel` không
		if err := c.BindJSON(&orderModel); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// validate.Struct() được sử dụng để kiểm tra xem một biến cấu trúc có đáp ứng được các quy tắc kiểm tra hợp lệ (validation rules) đã được xác định hay không.
		// Quy tắc kiểm tra này thường được định nghĩa thông qua các tag validate trong các trường của cấu trúc.
		validationErr := validate.Struct(orderModel)
		if validationErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
			return
		}

		// Kiểm tra `table_id` có tồn tại không
		if orderModel.Table_id != nil {
			// Tìm kiếm 1 tài liệu table từ bảng `table` với `table_id` từ request và kết quả trả về được tham chiếu tới tableModel
			err := tableCollection.FindOne(ctx, bson.M{"table_id": orderModel.Table_id}).Decode(&tableModel)
			defer cancel()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "table was not found"})
				return
			}
		}

		// Gán lại các giá trị khác
		orderModel.Created_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		orderModel.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

		orderModel.ID = primitive.NewObjectID()
		orderModel.Order_id = orderModel.ID.Hex()

		// Update lên mongo
		result, err := orderCollection.InsertOne(ctx, orderModel)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "order item was not created - " + err.Error()})
			return
		}
		defer cancel()

		c.JSON(http.StatusOK, result)
	}
}

func UpdateOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

		var orderModel models.Order
		var tableModel models.Table

		// Tạo 1 đối tượng update dạng primitive.D
		var updateObj primitive.D

		// Kiểm tra xem yêu cầu từ http có tham chiếu được tới `orderModel` không
		if err := c.BindJSON(&orderModel); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Set lại các giá trị
		if orderModel.Table_id != nil {
			// Tìm kiếm 1 tài liệu table từ bảng `table` với `table_id` từ request và kết quả trả về được tham chiếu tới tableModel
			err := tableCollection.FindOne(ctx, bson.M{"table_id": orderModel.Table_id}).Decode(&tableModel)
			defer cancel()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "table was not found"})
				return
			}
			updateObj = append(updateObj, bson.E{Key: "table_id", Value: orderModel.Table_id})
		}

		// Cập nhật lại `updated_at`
		orderModel.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		updateObj = append(updateObj, bson.E{Key: "updated_at", Value: orderModel.Updated_at})

		// Đây là một biến boolean được sử dụng để chỉ định xem truy vấn cập nhật có nên thực hiện một phép chèn mới (upsert) nếu không tìm thấy tài liệu phù hợp không.
		// Trong trường hợp này, giá trị true cho biết rằng upsert được kích hoạt.
		upsert := true
		// truy vấn cập nhật sẽ thực hiện một phép upsert nếu không tìm thấy tài liệu phù hợp vì đã set = true.
		opt := options.UpdateOptions{
			Upsert: &upsert,
		}

		// Lấy order_id từ request http
		var orderId = c.Param("order_id")
		// Tạo 1 bản ghi từ `food_id` bên trên để dùng làm giá trị filter
		filter := bson.M{"order_id": orderId}

		// update lại data trên mongo
		result, err := orderCollection.UpdateOne(
			ctx,
			filter,
			bson.D{{Key: "$set", Value: updateObj}},
			&opt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Order update failed - " + err.Error()})
			return
		}

		defer cancel()
		c.JSON(http.StatusOK, result)
	}
}

func orderItemCreator(ctx context.Context, cancel context.CancelFunc, orderModel models.Order) string {
	orderModel.Created_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
	orderModel.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
	orderModel.ID = primitive.NewObjectID()
	orderModel.Order_id = orderModel.ID.Hex()

	orderCollection.InsertOne(ctx, orderModel)
	defer cancel()

	return orderModel.Order_id
}

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

var tableCollection = database.OpenCollection(database.Client, "table")

func GetTables() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		// Find() được sử dụng để truy vấn tất cả các tài liệu trong bộ sưu tập.
		// Trong trường hợp này, context.TODO() được sử dụng để tạo một ngữ cảnh mặc định (context) không có thông tin bổ sung.
		// bson.M{} là một bộ lọc trống, chỉ đơn giản là yêu cầu tất cả các tài liệu.
		result, err := tableCollection.Find(context.TODO(), bson.M{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while listing table items"})
			return
		}

		var allTables []bson.M
		if err = result.All(ctx, &allTables); err != nil {
			log.Fatal(err)
			return
		}

		c.JSON(http.StatusOK, allTables)
	}
}

func GetTable() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()
		tableId := c.Param("table_id")
		var tableModel models.Table

		// Trả về 1 đối tượng table từ `table_id` được chỉ định, đối tượng nhận về được tham chiếu lại vào `tableModel`
		err := tableCollection.FindOne(ctx, bson.M{"table_id": tableId}).Decode(&tableModel)
		// Trả về lỗi nếu tồn tại
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while fetching the table item"})
			return
		}

		c.JSON(http.StatusOK, tableModel)
	}
}

func CreateTable() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var tableModel models.Table
		// Kiểm tra xem yêu cầu từ http có tham chiếu được tới `tableModel` không
		if err := c.BindJSON(&tableModel); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// validate.Struct() được sử dụng để kiểm tra xem một biến cấu trúc có đáp ứng được các quy tắc kiểm tra hợp lệ (validation rules) đã được xác định hay không.
		// Quy tắc kiểm tra này thường được định nghĩa thông qua các tag validate trong các trường của cấu trúc.
		validationErr := validate.Struct(tableModel)
		if validationErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
			return
		}

		// Set lại giá trị
		tableModel.Created_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		tableModel.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		tableModel.ID = primitive.NewObjectID()
		tableModel.Table_id = tableModel.ID.Hex()

		// Update lên mongo
		result, err := tableCollection.InsertOne(ctx, tableModel)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "table item was not created - " + err.Error()})
			return
		}
		defer cancel()

		c.JSON(http.StatusOK, result)
	}
}

func UpdateTable() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var tableModel models.Table
		// Kiểm tra xem yêu cầu từ http có tham chiếu được tới `tableModel` không
		if err := c.BindJSON(&tableModel); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Lấy table_id từ request
		tableId := c.Param("table_id")
		filter := bson.M{"table_id": tableId}

		// Tạo đối tượng update
		var updateObj primitive.D
		// Set giá trị
		if tableModel.Number_of_guests != nil {
			updateObj = append(updateObj, bson.E{Key: "number_of_guests", Value: *&tableModel.Number_of_guests})
		}
		if tableModel.Table_number != nil {
			updateObj = append(updateObj, bson.E{Key: "table_number", Value: *&tableModel.Table_number})
		}
		tableModel.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		updateObj = append(updateObj, bson.E{Key: "updated_at", Value: tableModel.Updated_at})

		// Đây là một biến boolean được sử dụng để chỉ định xem truy vấn cập nhật có nên thực hiện một phép chèn mới (upsert) nếu không tìm thấy tài liệu phù hợp không.
		// Trong trường hợp này, giá trị true cho biết rằng upsert được kích hoạt.
		upsert := true
		// truy vấn cập nhật sẽ thực hiện một phép upsert nếu không tìm thấy tài liệu phù hợp vì đã set = true.
		opt := options.UpdateOptions{
			Upsert: &upsert,
		}

		// Update lại giá trị trong mongo
		result, err := tableCollection.UpdateOne(
			ctx,
			filter,
			bson.D{{Key: "$set", Value: updateObj}},
			&opt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Table item update failed - " + err.Error()})
			return
		}

		defer cancel()
		c.JSON(http.StatusOK, result)
	}
}

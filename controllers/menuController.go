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
			return
		}

		c.JSON(http.StatusOK, allMenus)
	}
}

func GetMenu() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()
		menuId := c.Param("menu_id")
		var menuModel models.Menu

		// Trả về 1 đối tượng menu từ `menu_id` được chỉ định, đối tượng nhận về được tham chiếu lại vào `menuModel`
		err := menuCollection.FindOne(ctx, bson.M{"menu_id": menuId}).Decode(&menuModel)
		// Trả về lỗi nếu tồn tại
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while fetching the menu item"})
			return
		}

		c.JSON(http.StatusOK, menuModel)
	}
}

func CreateMenu() gin.HandlerFunc {
	return func(c *gin.Context) {
		var menuModel models.Menu
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		// Kiểm tra xem yêu cầu từ http có tham chiếu được tới `menuModel` không
		if err := c.BindJSON(&menuModel); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// validate.Struct() được sử dụng để kiểm tra xem một biến cấu trúc có đáp ứng được các quy tắc kiểm tra hợp lệ (validation rules) đã được xác định hay không.
		// Quy tắc kiểm tra này thường được định nghĩa thông qua các tag validate trong các trường của cấu trúc.
		validationErr := validate.Struct(menuModel)
		if validationErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
			return
		}

		// Gán lại các giá trị khác
		menuModel.Created_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		menuModel.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		menuModel.ID = primitive.NewObjectID()
		menuModel.Menu_id = menuModel.ID.Hex()

		// Insert menuModel vào bảng `menu`
		result, insertErr := menuCollection.InsertOne(ctx, menuModel)
		if insertErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "menu item was not created"})
			return
		}
		defer cancel()

		c.JSON(http.StatusOK, result)
	}
}

func UpdateMenu() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		var menuModel models.Menu

		// Kiểm tra xem yêu cầu từ http có tham chiếu được tới `menuModel` không
		if err := c.BindJSON(&menuModel); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Lấy tham số `menu_id` từ request http
		menuId := c.Param("menu_id")
		// Tạo 1 bản ghi từ `menu_id` bên trên để dùng làm giá trị filter
		filter := bson.M{"menu_id": menuId}

		// primitive.D là một kiểu dữ liệu được sử dụng để đại diện cho một tài liệu BSON (Binary JSON) dưới dạng danh sách các cặp khóa-giá trị.
		// Nó khác primitive.D cũng tạo ra kiểu khóa giá trị nhưng nó sử dụng ở dạng map
		/*
						doc := primitive.D{
			    			{"name", "John Doe"},
			    			{"age", 30},
			    			{"email", "john@example.com"},
						}
		*/
		var updateObj primitive.D

		if menuModel.Start_date != nil && menuModel.End_date != nil {
			if !inTimeSpan(*menuModel.Start_date, *menuModel.End_date, time.Now()) {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "kindly retype the time"})
				defer cancel()
				return
			}

			// bson.E là một kiểu dữ liệu được sử dụng để biểu diễn một cặp khóa-giá trị trong một tài liệu BSON (Binary JSON).
			updateObj = append(updateObj, bson.E{Key: "start_date", Value: menuModel.Start_date})
			updateObj = append(updateObj, bson.E{Key: "end_date", Value: menuModel.End_date})

			if menuModel.Name != "" {
				updateObj = append(updateObj, bson.E{Key: "name", Value: menuModel.Name})
			}
			if menuModel.Category != "" {
				updateObj = append(updateObj, bson.E{Key: "category", Value: menuModel.Category})
			}

			// Cập nhật lại `created_at`
			menuModel.Created_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
			updateObj = append(updateObj, bson.E{Key: "created_at", Value: menuModel.Created_at})
			// Cập nhật lại `updated_at`
			menuModel.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
			updateObj = append(updateObj, bson.E{Key: "updated_at", Value: menuModel.Updated_at})

			// Đây là một biến boolean được sử dụng để chỉ định xem truy vấn cập nhật có nên thực hiện một phép chèn mới (upsert) nếu không tìm thấy tài liệu phù hợp không.
			// Trong trường hợp này, giá trị true cho biết rằng upsert được kích hoạt.
			upsert := true
			// truy vấn cập nhật sẽ thực hiện một phép upsert nếu không tìm thấy tài liệu phù hợp vì đã set = true.
			opt := options.UpdateOptions{
				Upsert: &upsert,
			}

			// Update lại giá trị trong mongo
			/*
				Key: "$set": Đây là một cặp khóa-giá trị trong tài liệu BSON.
				Trong trường hợp này, khóa là "$set", là một toán tử cập nhật trong MongoDB,
				chỉ định rằng các trường cần được cập nhật sẽ được chỉ định bằng các giá trị trong updateObj.
			*/
			result, err := menuCollection.UpdateOne(
				ctx,
				filter,
				bson.D{{Key: "$set", Value: updateObj}},
				&opt,
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Menu update failed - " + err.Error()})
				return
			}

			defer cancel()
			c.JSON(http.StatusOK, result)
		}
	}
}

// Kiểm tra xem thời gian `check` có nằm trong khoảng thời gian start và end không
func inTimeSpan(start, end, check time.Time) bool {
	return start.After(check) && end.After(start)
}

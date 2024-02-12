package controllers

import (
	"context"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/rongdo4897/restaurant-manager-go/database"
	"github.com/rongdo4897/restaurant-manager-go/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Kết nối tới bảng `food`
var foodCollection = database.OpenCollection(database.Client, "food")

// Tạo 1 đối tượng Validate
var validate = validator.New()

func GetFoods() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)

		// Đọc giá trị `recordPerPage` từ request http và chuyển nó thành giá trị int, nó tương đương với giá trị count
		// Việc xử lý như này phục vụ cho việc phân trang dữ liệu
		recordPerPage, err := strconv.Atoi(c.Query("recordPerPage"))
		if err != nil || recordPerPage < 1 {
			recordPerPage = 10
		}

		// Đọc giá trị `page` từ request http và chuyển nó thành giá trị int,
		page, err := strconv.Atoi(c.Query("page"))
		if err != nil || page < 1 {
			page = 1
		}

		// Tính toán vị trí bắt đầu để lấy dữ liệu
		startIndex := (page - 1) * recordPerPage
		// lấy giá trị bắt đầu từ `startIndex` dựa trên request http
		// startIndex, err = strconv.Atoi(c.Query("startIndex"))

		/*
			$match là một toán tử aggregation được sử dụng để lọc các tài liệu từ bộ sưu tập dựa trên các điều kiện cụ thể.
			Value: bson.D{{}} chỉ định rằng không có điều kiện lọc cụ thể được áp dụng và tất cả các tài liệu sẽ được trả về.
			=> `matchStage`` sẽ trả về tất cả tài liệu
		*/
		matchStage := bson.D{{Key: "$match", Value: bson.D{{}}}}
		// `groupStage` chịu trách nhiệm nhóm các tài liệu dựa trên các điều kiện cụ thể
		/*
			Điều kiện 1:
				"$group" được sử dụng để nhóm các tài liệu theo các điều kiện nhất định
				"_id": Đây là trường được sử dụng để xác định các nhóm trong quá trình nhóm.
				"_id": "null": Trường _id được đặt thành "null" để chỉ định rằng tất cả các tài liệu sẽ được nhóm vào một nhóm duy nhất.

			Điều kiện 2:
				"total_count": Đây là tên của một trường mới sẽ được tạo trong kết quả đầu ra, đại diện cho tổng số tài liệu trong mỗi nhóm.
				"$sum": Đây là toán tử aggregation $sum của MongoDB, được sử dụng để tính tổng của các giá trị.
				`1`: Đây là giá trị được sử dụng để tính tổng. Trong trường hợp này, mỗi tài liệu sẽ được tính là 1.

			Điều kiện 3:
				"data": Đây là tên của một trường mới sẽ được tạo trong kết quả đầu ra, đại diện cho dữ liệu trong mỗi nhóm.
				"$push": Đây là toán tử aggregation $push của MongoDB, được sử dụng để thêm các giá trị vào một mảng.
				"$$ROOT": Đây là biến tham chiếu đến toàn bộ tài liệu trong mỗi nhóm. Trong trường hợp này, mọi trường của tài liệu sẽ được thêm vào một mảng.

			=> `groupStage` sẽ chứa một giai đoạn $group trong truy vấn aggregation của MongoDB,
				trong đó các tài liệu sẽ được nhóm thành một nhóm duy nhất,
				với trường `total_count` biểu diễn tổng số tài liệu trong nhóm và trường `data` chứa tất cả các tài liệu trong nhóm.
		*/
		groupStage := bson.D{
			{Key: "$group", Value: bson.D{{Key: "_id", Value: bson.D{{Key: "_id", Value: "null"}}}}},
			{Key: "total_count", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "data", Value: bson.D{{Key: "$push", Value: "$$ROOT"}}},
		}
		// `projectStage` để chọn lọc các trường từ kết quả truy vấn và chỉ trả về các trường được chỉ định
		/*
			Key: "$project": được sử dụng để chọn lọc các trường từ kết quả truy vấn và chỉ trả về các trường được chỉ định.

			Có 3 trường giá trị được chọn lọc:
				- {Key: "_id", Value: 0}: Trường _id được loại bỏ khỏi kết quả đầu ra (có giá trị là 0).
				- {Key: "total_count", Value: 1}: Trường total_count sẽ được bao gồm trong kết quả đầu ra (có giá trị là 1), biểu diễn tổng số tài liệu trong nhóm.
				- {Key: "food_items", Value: bson.D{{Key: "$slice", Value: []interface{}{"$data", startIndex, recordPerPage}}}}:
					+ Key: "food_items": Đây là tên của trường mới sẽ được tạo trong kết quả đầu ra, đại diện cho các mục thực phẩm.
					+ Value: bson.D{{Key: "$slice", Value: []interface{}{"$data", startIndex, recordPerPage}}}:
					  	Đây là một truy vấn để lấy một phần của mảng data, đại diện cho các mục thực phẩm.
					  	Trong trường hợp này, $slice là một toán tử aggregation của MongoDB được sử dụng để trích xuất một phần của mảng.
							. "data" là tên của trường mảng cần được trích xuất.
							. startIndex là chỉ số bắt đầu của phần được trích xuất.
							. recordPerPage là số lượng phần tử cần trích xuất từ startIndex.

			=> projectStage sẽ chứa một giai đoạn $project trong truy vấn aggregation của MongoDB,
			   trong đó các trường được chỉ định sẽ được bao gồm trong kết quả đầu ra,
			   bao gồm tổng số tài liệu trong nhóm (total_count) và các mục thực phẩm (food_items) được trích xuất từ mảng data.
		*/
		projectStage := bson.D{
			{
				Key: "$project",
				Value: bson.D{
					{Key: "_id", Value: 0},
					{Key: "total_count", Value: 1},
					{Key: "food_items", Value: bson.D{{Key: "$slice", Value: []interface{}{"$data", startIndex, recordPerPage}}}},
				},
			},
		}

		// truy vấn aggregation trên một bộ sưu tập trong MongoDB, sử dụng các giai đoạn (stages) đã được xác định trước đó.
		result, err := foodCollection.Aggregate(ctx, mongo.Pipeline{
			matchStage, groupStage, projectStage,
		})
		defer cancel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Can't get listing food items - " + err.Error()})
			return
		}

		// Chuyển đổi dữ liệu sang mảng bson.M
		var allFoods []bson.M
		if err := result.All(ctx, &allFoods); err != nil {
			log.Fatalln(err)
			return
		}

		c.JSON(http.StatusOK, allFoods[0])
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
			return
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
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		var menuModel models.Menu
		var foodModel models.Food

		// Lấy param `food_id` từ request
		food_id := c.Param("food_id")

		// Kiểm tra xem yêu cầu từ http có tham chiếu được tới `foodModel` không
		if err := c.BindJSON(&foodModel); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Tạo biến dạng danh sách Key-Value để phục vụ cho update data
		var updateObj primitive.D

		// Set các giá trị
		if foodModel.Name != nil {
			updateObj = append(updateObj, bson.E{Key: "name", Value: foodModel.Name})
		}

		if foodModel.Price != nil {
			updateObj = append(updateObj, bson.E{Key: "price", Value: foodModel.Price})
		}

		if foodModel.Food_image != nil {
			updateObj = append(updateObj, bson.E{Key: "food_image", Value: foodModel.Food_image})
		}

		if foodModel.Menu_id != nil {
			err := menuCollection.FindOne(ctx, bson.M{"menu_id": foodModel.Menu_id}).Decode(&menuModel)
			defer cancel()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "menu was not found"})
				return
			}
			updateObj = append(updateObj, bson.E{Key: "menu_id", Value: foodModel.Menu_id})
		}

		// Cập nhật lại `updated_at`
		foodModel.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		updateObj = append(updateObj, bson.E{Key: "updated_at", Value: foodModel.Updated_at})

		// Đây là một biến boolean được sử dụng để chỉ định xem truy vấn cập nhật có nên thực hiện một phép chèn mới (upsert) nếu không tìm thấy tài liệu phù hợp không.
		// Trong trường hợp này, giá trị true cho biết rằng upsert được kích hoạt.
		upsert := true
		// truy vấn cập nhật sẽ thực hiện một phép upsert nếu không tìm thấy tài liệu phù hợp vì đã set = true.
		opt := options.UpdateOptions{
			Upsert: &upsert,
		}

		// Tạo 1 bản ghi từ `food_id` bên trên để dùng làm giá trị filter
		filter := bson.M{"food_id": food_id}

		// update lại data trên mongo
		result, err := foodCollection.UpdateOne(
			ctx,
			filter,
			bson.D{{Key: "$set", Value: updateObj}},
			&opt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Food update failed - " + err.Error()})
			return
		}

		defer cancel()
		c.JSON(http.StatusOK, result)
	}
}

// Làm tròn số thập phân thành số nguyên
func round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

// Làm tròn 1 số thập phân đến một số lượng chữ số thập phân cụ thể
func toFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(round(num*output)) / output
}

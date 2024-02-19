package controllers

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rongdo4897/restaurant-manager-go/database"
	"github.com/rongdo4897/restaurant-manager-go/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var userCollection = database.OpenCollection(database.Client, "user")

func GetUsers() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		// Lấy số lượng bản ghi trong 1 page `recordPerPage` từ request
		recordPerPage, err := strconv.Atoi(c.Query("recordPerPage"))
		if err != nil || recordPerPage < 1 {
			recordPerPage = 10
		}

		// Lấy giá trị page từ request
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
		// `projectStage` để chọn lọc các trường từ kết quả truy vấn và chỉ trả về các trường được chỉ định
		/*
			Key: "$project": được sử dụng để chọn lọc các trường từ kết quả truy vấn và chỉ trả về các trường được chỉ định.

			Có 3 trường giá trị được chọn lọc:
				- {Key: "_id", Value: 0}: Trường _id được loại bỏ khỏi kết quả đầu ra (có giá trị là 0).
				- {Key: "total_count", Value: 1}: Trường total_count sẽ được bao gồm trong kết quả đầu ra (có giá trị là 1), biểu diễn tổng số tài liệu trong nhóm.
				- {Key: "user_items", Value: bson.D{{Key: "$slice", Value: []interface{}{"$data", startIndex, recordPerPage}}}}:
					+ Key: "user_items": Đây là tên của trường mới sẽ được tạo trong kết quả đầu ra, đại diện cho các mục thực phẩm.
					+ Value: bson.D{{Key: "$slice", Value: []interface{}{"$data", startIndex, recordPerPage}}}:
					  	Đây là một truy vấn để lấy một phần của mảng data, đại diện cho các mục thực phẩm.
					  	Trong trường hợp này, $slice là một toán tử aggregation của MongoDB được sử dụng để trích xuất một phần của mảng.
							. "data" là tên của trường mảng cần được trích xuất.
							. startIndex là chỉ số bắt đầu của phần được trích xuất.
							. recordPerPage là số lượng phần tử cần trích xuất từ startIndex.

			=> projectStage sẽ chứa một giai đoạn $project trong truy vấn aggregation của MongoDB,
			   trong đó các trường được chỉ định sẽ được bao gồm trong kết quả đầu ra,
			   bao gồm tổng số tài liệu trong nhóm (total_count) và các mục thực phẩm (user_items) được trích xuất từ mảng data.
		*/
		projectStage := bson.D{
			{
				Key: "$project",
				Value: bson.D{
					{Key: "_id", Value: 0},
					{Key: "total_count", Value: 1},
					{Key: "user_items", Value: bson.D{{Key: "$slice", Value: []interface{}{"$data", startIndex, recordPerPage}}}},
				},
			},
		}

		// truy vấn aggregation trên một bộ sưu tập trong MongoDB, sử dụng các giai đoạn (stages) đã được xác định trước đó.
		result, err := userCollection.Aggregate(ctx, mongo.Pipeline{
			matchStage, projectStage,
		})
		defer cancel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Can't get listing user items - " + err.Error()})
			return
		}

		var allUsers []models.User
		if err := result.All(ctx, &allUsers); err != nil {
			log.Fatalln(err)
			return
		}

		c.JSON(http.StatusOK, allUsers[0])
	}
}

func GetUser() gin.HandlerFunc {
	return func(c *gin.Context) {

	}
}

func SignUp() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Convert the JSON data coming from postman to something that golang understands

		// Validate the data based on user struct

		// You will check if the email has already been used by another user

		// Hash password

		// You will also check if the phone number has already been used by another user

		// Create some extra details for the user object - created_at, updated_at, ID

		// Generate token and refresh token (generate all tokens function from helpers)

		// If all ok, then you insert this new user into the user collection

		// return status ok and send the result back
	}
}

func Login() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Convert the login data from postman which is in JSON to golang readable format

		// Find a user with that email and see if that user even exists

		// Then you will verify the password

		// If all goes well, then you will generate tokens

		// Update tokens - token, refreshToken

		// Return
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

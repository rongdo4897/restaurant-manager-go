package controllers

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rongdo4897/restaurant-manager-go/database"
	"github.com/rongdo4897/restaurant-manager-go/helpers"
	"github.com/rongdo4897/restaurant-manager-go/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
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
					{Key: "_id", Value: 1},
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

		c.JSON(http.StatusOK, allUsers)
	}
}

func GetUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()
		userId := c.Param("user_id")
		var userModel models.User

		// Trả về 1 đối tượng food từ `user_id` được chỉ định, đối tượng nhận về được tham chiếu lại vào `userModel`
		err := userCollection.FindOne(ctx, bson.M{"user_id": userId}).Decode(&userModel)
		// Trả về lỗi nếu tồn tại
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while fetching the user item"})
			return
		}

		c.JSON(http.StatusOK, userModel)
	}
}

func SignUp() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var userModel models.User
		// Chuyển đổi request sang userModel
		if err := c.BindJSON(&userModel); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// Kiểm tra xem dữ liệu đã bao gồm các trường validate require chưa
		if validationErr := validate.Struct(userModel); validationErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
			return
		}
		// Kiểm tra xem email đã được người dùng khác sử dụng chưa
		countEmail, err := userCollection.CountDocuments(ctx, bson.M{"email": userModel.Email})
		if err != nil {
			log.Panic(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while checking for the email"})
			return
		}
		// Băm mật khẩu
		password := HashPassword(*userModel.Password)
		userModel.Password = &password
		// Kiểm tra xem phone number đã được người dùng khác sử dụng chưa
		countPhone, err := userCollection.CountDocuments(ctx, bson.M{"phone": userModel.Phone})
		if err != nil {
			log.Panic(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while checking for the phone number"})
			return
		}

		// Kiểm tra xem nếu count > 0 không (tồn tại)
		if countEmail > 0 || countPhone > 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "this email or phone already exists"})
			return
		}
		// Gán lại các giá trị khác
		userModel.Created_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		userModel.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		userModel.ID = primitive.NewObjectID()
		userModel.User_id = userModel.ID.Hex()
		// Tạo token và refresh token (generate all tokens function from helpers)
		token, refreshToken, _ := helpers.GenerateAllTokens(*userModel.Email, *userModel.First_name, *userModel.Last_name, userModel.User_id)
		userModel.Token = &token
		userModel.Refresh_token = &refreshToken

		// Insert vào mongo
		result, err := userCollection.InsertOne(ctx, userModel)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "User item was not created - " + err.Error()})
			return
		}
		defer cancel()

		c.JSON(http.StatusOK, result)
	}
}

func Login() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var userModel models.User
		var foundUserModel models.User
		// Chuyển đổi request sang userModel
		if err := c.BindJSON(&userModel); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// Tìm kiếm user trên database với email
		err := userCollection.FindOne(ctx, bson.M{"email": userModel.Email}).Decode(&foundUserModel)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "User not found, login seems to be incorrect - " + err.Error()})
			return
		}
		// Xác thực mật khẩu
		passwordIsValid, msg := VerifyPassword(*userModel.Password, *foundUserModel.Password)
		if !passwordIsValid {
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}
		// Tạo token và refresh token (generate all tokens function from helpers)
		token, refreshToken, _ := helpers.GenerateAllTokens(*foundUserModel.Email, *foundUserModel.First_name, *foundUserModel.Last_name, foundUserModel.User_id)
		// Update lại tokens - token, refreshToken
		helpers.UpdateAllTokens(token, refreshToken, foundUserModel.User_id)
		// Return
		defer cancel()

		c.JSON(http.StatusOK, foundUserModel)
	}
}

// Chuyển đổi mật khẩu đầu vào thành 1 chuỗi không thể bị đảo ngược
func HashPassword(password string) string {
	// Tạo ra 1 hash từ mật khẩu người dùng
	// Tham số đầu tiên []byte(password) là mật khẩu của người dùng, được chuyển đổi thành một mảng byte.
	// Tham số thứ hai 14 là cost factor, cũng gọi là work factor, quyết định độ phức tạp của thuật toán bcrypt.
	// Độ phức tạp này càng cao thì việc tạo ra hash càng mất thời gian, từ đó làm cho việc tìm kiếm bằng cách thử từng giá trị hash trở nên khó khăn hơn đối với các kỹ thuật tấn công brute force hoặc dictionary attack.
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		log.Panic(err)
	}

	return string(bytes)
}

// So sánh xác thực mật khẩu
func VerifyPassword(userPassword, providePassword string) (bool, string) {
	// So sánh giữa password được cung cấp và password người dùng
	err := bcrypt.CompareHashAndPassword([]byte(providePassword), []byte(userPassword))

	check := true
	msg := ""

	if err != nil {
		msg = "Login with password is incorrect"
		check = false
	}

	return check, msg
}

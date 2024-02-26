package helpers

import (
	"context"
	"log"
	"os"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/rongdo4897/restaurant-manager-go/database"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type SignedDetails struct {
	Email      string
	First_name string
	Last_name  string
	Uid        string
	jwt.StandardClaims
}

var userCollection = database.OpenCollection(database.Client, "user")

var SECRET_KEY = os.Getenv("SECRET_KEY")

func GenerateAllTokens(email, firstName, lastName, userId string) (string, string, error) {
	/*
		- claims là một thể hiện của cấu trúc SignedDetails. Cấu trúc này chứa các thông tin mà bạn muốn mã hóa và nhúng vào JWT (JSON Web Token) sau khi ký.
		- claims bao gồm các trường sau:
			+ Email: Địa chỉ email của người dùng.
			+ First_name: Tên của người dùng.
			+ Last_name: Họ của người dùng.
			+ Uid: Mã định danh của người dùng.
			+ StandardClaims: Một cấu trúc con của jwt.StandardClaims, một phần của thư viện JWT, chứa thông tin chuẩn cho JWT như thời gian hết hạn (ExpiresAt).

		- Trong đoạn mã trên, thời gian hết hạn của JWT được đặt là thời điểm hiện tại cộng với 24 giờ,
			sử dụng phương thức time.Now().Local().Add(time.Hour * time.Duration(24)).
			Sau đó, Unix() được gọi để chuyển đổi thời gian thành định dạng Unix epoch (tính bằng số giây kể từ 1/1/1970).

		=> claims đóng vai trò là dữ liệu được mã hóa và nhúng vào JWT, bao gồm thông tin về người dùng và thời gian hết hạn của token.
	*/
	claims := &SignedDetails{
		Email:      email,
		First_name: firstName,
		Last_name:  lastName,
		Uid:        userId,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Local().Add(time.Hour * time.Duration(24)).Unix(),
		},
	}

	refreshClaims := &SignedDetails{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Local().Add(time.Hour * time.Duration(168)).Unix(),
		},
	}

	/*
		- tạo ra một chuỗi token JWT được ký và mã hóa từ các thông tin trong biến claims sử dụng phương thức ký jwt.SigningMethodES256.
			Sau đó, chuỗi token được trả về cùng với một giá trị lỗi (nếu có) từ hàm SignedString.
			+ jwt.NewWithClaims(jwt.SigningMethodES256, claims): Tạo một đối tượng JWT mới với phương thức ký là jwt.SigningMethodES256 (ECDSA with SHA-256),
				và các thông tin được mã hóa từ biến claims.
			+ .SignedString([]byte(SECRET_KEY)): Sử dụng khóa bí mật (SECRET_KEY) để ký và mã hóa chuỗi token. Kết quả là chuỗi token đã được ký và mã hóa.
	*/
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(SECRET_KEY))
	if err != nil {
		log.Panic(err)
		return "", "", err
	}

	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString([]byte(SECRET_KEY))
	if err != nil {
		log.Panic(err)
		return "", "", err
	}

	return token, refreshToken, err
}

func UpdateAllTokens(token, refreshToken, userId string) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

	var updateObj primitive.D

	// Cập nhật đối tượng update
	updateObj = append(updateObj, bson.E{Key: "token", Value: token})
	updateObj = append(updateObj, bson.E{Key: "refresh_token", Value: refreshToken})

	update_at, _ := time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
	updateObj = append(updateObj, bson.E{Key: "updated_at", Value: update_at})

	// Đây là một biến boolean được sử dụng để chỉ định xem truy vấn cập nhật có nên thực hiện một phép chèn mới (upsert) nếu không tìm thấy tài liệu phù hợp không.
	// Trong trường hợp này, giá trị true cho biết rằng upsert được kích hoạt.
	upsert := true
	// truy vấn cập nhật sẽ thực hiện một phép upsert nếu không tìm thấy tài liệu phù hợp vì đã set = true.
	opt := options.UpdateOptions{
		Upsert: &upsert,
	}

	// Tạo 1 bản ghi từ `food_id` bên trên để dùng làm giá trị filter
	filter := bson.M{"user_id": userId}

	// Cập nhật lại mongo
	_, err := userCollection.UpdateOne(
		ctx,
		filter,
		bson.D{{Key: "$set", Value: updateObj}},
		&opt,
	)
	if err != nil {
		log.Fatal(err)
		return
	}
}

func ValidateToken(signedToken string) (claims *SignedDetails, msg string) {
	token, err := jwt.ParseWithClaims(
		signedToken,
		&SignedDetails{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(SECRET_KEY), nil
		},
	)

	if err != nil {
		return nil, err.Error()
	}

	// Token không hợp lệ
	claims, ok := token.Claims.(*SignedDetails)
	if !ok {
		msg = "Token is invalid"
	}

	// Token hết hạn
	if claims.ExpiresAt < time.Now().Local().Unix() {
		msg = "Token is expired"
	}

	return claims, msg
}

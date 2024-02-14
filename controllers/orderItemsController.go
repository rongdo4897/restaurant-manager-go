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
)

var orderItemCollection = database.OpenCollection(database.Client, "orderItem")

type OrderItemPack struct {
	Table_id    *string
	Order_items []models.OrderItem
}

func GetOrderItems() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		// Find() được sử dụng để truy vấn tất cả các tài liệu trong bộ sưu tập.
		// Trong trường hợp này, context.TODO() được sử dụng để tạo một ngữ cảnh mặc định (context) không có thông tin bổ sung.
		// bson.M{} là một bộ lọc trống, chỉ đơn giản là yêu cầu tất cả các tài liệu.
		result, err := orderItemCollection.Find(context.TODO(), bson.M{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while listing order items"})
			return
		}

		var allOrderItems []bson.M
		if err = result.All(ctx, &allOrderItems); err != nil {
			log.Fatal(err)
			return
		}

		c.JSON(http.StatusOK, allOrderItems)
	}
}

func GetOrderItem() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		order_item_id := c.Param("order_item_id")
		var orderItemModel models.OrderItem

		// Lấy data dựa trên `order_item_id`
		err := orderItemCollection.FindOne(ctx, bson.M{"order_item_id": order_item_id}).Decode(orderItemModel)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while fetching the order item"})
			return
		}

		c.JSON(http.StatusOK, orderItemModel)
	}
}

func GetOrderItemsByOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		order_id := c.Param("order_id")

		// Lấy danh sách order item dựa trên `order_id` được cung cấp từ request
		// 1 order có thể bao gồm nhiều item
		allOrderItems, err := itemByOrder(ctx, cancel, order_id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while listing order items by order id"})
			return
		}

		c.JSON(http.StatusOK, allOrderItems)
	}
}

func CreateOrderItem() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var orderItemPack OrderItemPack
		var orderModel models.Order

		// Kiểm tra request gửi lên có map với kiểu `OrderItemPack` không
		if err := c.BindJSON(&orderItemPack); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Tạo danh sách dữ liệu OrderItem được khởi tạo
		orderItemsToBeInserted := []interface{}{}

		// Set giá trị
		orderModel.Order_date, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		orderModel.Table_id = orderItemPack.Table_id

		// Lấy `order_id` dựa trên orderModel
		order_id := orderItemOrderCreator(orderModel)

		for _, orderItem := range orderItemPack.Order_items {
			orderItem.Order_id = order_id

			// Validate kiểu dữ liệu đầu vào OrderItem
			validateErr := validate.Struct(orderItem)
			if validateErr != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": validateErr.Error()})
				return
			}

			// Gán các giá trị cho OrderItem
			orderItem.ID = primitive.NewObjectID()
			orderItem.Created_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
			orderItem.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
			orderItem.Order_item_id = orderItem.ID.Hex()
			var number = toFixed(*orderItem.Unit_price, 2)
			orderItem.Unit_price = &number

			// Append OrderItem vào mảng
			orderItemsToBeInserted = append(orderItemsToBeInserted, orderItem)
		}

		// Thêm dữ liệu danh sách OrderItem bên trên vào mongo
		insertOrderItemResult, err := orderItemCollection.InsertMany(ctx, orderItemsToBeInserted)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Insert list order items failed - " + err.Error()})
			return
		}
		defer cancel()

		c.JSON(http.StatusOK, insertOrderItemResult)
	}
}

func UpdateOrderItem() gin.HandlerFunc {
	return func(c *gin.Context) {

	}
}

func itemByOrder(ctx context.Context, cancel context.CancelFunc, id string) (orderItems []primitive.M, err error) {
	return make([]primitive.M, 0), nil
}

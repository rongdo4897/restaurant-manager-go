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

type InvoiceViewFormat struct {
	Invoice_id       string
	Payment_method   string
	Order_id         string
	Payment_status   *string
	Payment_due      interface{}
	Table_number     interface{}
	Payment_due_date time.Time
	Order_details    interface{}
}

var invoiceCollection = database.OpenCollection(database.Client, "invoice")

func GetInvoices() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		// Find() được sử dụng để truy vấn tất cả các tài liệu trong bộ sưu tập.
		// Trong trường hợp này, context.TODO() được sử dụng để tạo một ngữ cảnh mặc định (context) không có thông tin bổ sung.
		// bson.M{} là một bộ lọc trống, chỉ đơn giản là yêu cầu tất cả các tài liệu.
		result, err := invoiceCollection.Find(context.TODO(), bson.M{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while listing invoice items"})
			return
		}

		var allInvoices []bson.M
		if err = result.All(ctx, &allInvoices); err != nil {
			log.Fatal(err)
		}

		c.JSON(http.StatusOK, allInvoices)
	}
}

func GetInvoice() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		invoice_id := c.Param("invoice_id")
		var invoiceModel models.Invoice

		// Trả về 1 đối tượng invoice từ `invoice_id` được chỉ định, đối tượng nhận về được tham chiếu lại vào `invoiceModel`
		err := invoiceCollection.FindOne(ctx, bson.M{"invoice_id": invoice_id}).Decode(&invoiceModel)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while fetching the invoice item"})
			return
		}

		// Chuyển đổi dữ liệu invoice được chỉ định
		var invoiceView InvoiceViewFormat

		// Trả về danh sách item dựa trên `order_id`
		allOrderItems, err := itemByOrder(invoiceModel.Order_id)
		if err != nil {
			log.Fatal(err)
			return
		}

		invoiceView.Order_id = invoiceModel.Order_id
		invoiceView.Payment_due_date = invoiceModel.Payment_due_date
		// Gán lại `Payment_method`
		invoiceView.Payment_method = "null"
		if invoiceModel.Payment_method != nil {
			invoiceView.Payment_method = *invoiceModel.Payment_method
		}
		// Gán lại các giá trị khác
		invoiceView.Invoice_id = invoiceModel.Invoice_id
		invoiceView.Payment_status = *&invoiceModel.Payment_status
		invoiceView.Payment_due = allOrderItems[0]["payment_due"]
		invoiceView.Table_number = allOrderItems[0]["table_number"]
		invoiceView.Order_details = allOrderItems[0]["order_items"]

		// Trả về kết quả
		c.JSON(http.StatusOK, invoiceView)
	}
}

func CreateInvoice() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var invoiceModel models.Invoice
		var orderModel models.Order

		// Kiểm tra xem yêu cầu từ http có tham chiếu được tới `invoiceModel` không
		if err := c.BindJSON(&invoiceModel); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Kiểm tra xem `order_id` có tồn tại không và dữ liệu từ bảng `order` có tham chiếu được tới `orderModel` không
		err := orderCollection.FindOne(ctx, bson.M{"order_id": invoiceModel.Order_id}).Decode(&orderModel)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "order not found"})
			return
		}

		// Set các giá trị
		if invoiceModel.Payment_status == nil {
			status := "PENDING"
			invoiceModel.Payment_status = &status
		}

		invoiceModel.Payment_due_date, _ = time.Parse(time.RFC3339, time.Now().AddDate(0, 0, 1).Format(time.RFC3339))
		invoiceModel.Created_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		invoiceModel.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		invoiceModel.ID = primitive.NewObjectID()
		invoiceModel.Invoice_id = invoiceModel.ID.Hex()

		// Kiểm tra validate
		validationErr := validate.Struct(invoiceModel)
		if validationErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
			return
		}

		// Update lên mongo
		result, err := invoiceCollection.InsertOne(ctx, invoiceModel)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invoice item was not created - " + err.Error()})
			return
		}
		defer cancel()

		c.JSON(http.StatusOK, result)
	}
}

func UpdateInvoice() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var invoiceModel models.Invoice

		// Kiểm tra xem yêu cầu từ http có tham chiếu được tới `orderModel` không
		if err := c.BindJSON(&invoiceModel); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		invoice_id := c.Param("invoice_id")
		filter := bson.M{"invoice_id": invoice_id}

		// Tạo đối tượng update
		var updateObj primitive.D

		// Set các giá trị
		if invoiceModel.Payment_method != nil {
			updateObj = append(updateObj, bson.E{Key: "payment_method", Value: invoiceModel.Payment_method})
		}

		if invoiceModel.Payment_status != nil {
			updateObj = append(updateObj, bson.E{Key: "payment_status", Value: invoiceModel.Payment_status})
		} else {
			status := "PENDING"
			invoiceModel.Payment_status = &status
			updateObj = append(updateObj, bson.E{Key: "payment_status", Value: invoiceModel.Payment_status})
		}

		// Cập nhật lại `updated_at`
		invoiceModel.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		updateObj = append(updateObj, bson.E{Key: "updated_at", Value: invoiceModel.Updated_at})

		// Đây là một biến boolean được sử dụng để chỉ định xem truy vấn cập nhật có nên thực hiện một phép chèn mới (upsert) nếu không tìm thấy tài liệu phù hợp không.
		// Trong trường hợp này, giá trị true cho biết rằng upsert được kích hoạt.
		upsert := true
		// truy vấn cập nhật sẽ thực hiện một phép upsert nếu không tìm thấy tài liệu phù hợp vì đã set = true.
		opt := options.UpdateOptions{
			Upsert: &upsert,
		}

		// update lại data trên mongo
		result, err := orderCollection.UpdateOne(
			ctx,
			filter,
			bson.D{{Key: "$set", Value: updateObj}},
			&opt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invoice update failed - " + err.Error()})
			return
		}

		defer cancel()
		c.JSON(http.StatusOK, result)
	}
}

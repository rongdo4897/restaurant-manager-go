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

	}
}

func UpdateInvoice() gin.HandlerFunc {
	return func(c *gin.Context) {

	}
}

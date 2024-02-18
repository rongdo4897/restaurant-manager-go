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
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var orderItemModel models.OrderItem
		// Kiểm tra xem yêu cầu từ http có tham chiếu được tới `orderItemModel` không
		if err := c.BindJSON(&orderItemModel); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Lấy `order_item_id` từ request
		orderItemId := c.Param("order_item_id")
		// Tạo giá trị cho bộ lọc
		filter := bson.M{"order_item_id": orderItemId}

		// Tạo biến cho việc update
		var updateObj primitive.D

		// Set lại giá trị
		if orderItemModel.Unit_price != nil {
			updateObj = append(updateObj, bson.E{Key: "unit_price", Value: *&orderItemModel.Unit_price})
		}
		if orderItemModel.Quantity != nil {
			updateObj = append(updateObj, bson.E{Key: "quantity", Value: *orderItemModel.Quantity})
		}
		if orderItemModel.Food_id != nil {
			updateObj = append(updateObj, bson.E{Key: "food_id", Value: *orderItemModel.Food_id})
		}
		orderItemModel.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		updateObj = append(updateObj, bson.E{Key: "updated_at", Value: orderItemModel.Updated_at})

		// Đây là một biến boolean được sử dụng để chỉ định xem truy vấn cập nhật có nên thực hiện một phép chèn mới (upsert) nếu không tìm thấy tài liệu phù hợp không.
		// Trong trường hợp này, giá trị true cho biết rằng upsert được kích hoạt.
		upsert := true
		// truy vấn cập nhật sẽ thực hiện một phép upsert nếu không tìm thấy tài liệu phù hợp vì đã set = true.
		opt := options.UpdateOptions{
			Upsert: &upsert,
		}

		// Update lại giá trị trong mongo
		result, err := orderItemCollection.UpdateOne(
			ctx,
			filter,
			bson.D{{Key: "$set", Value: updateObj}},
			&opt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Order item update failed - " + err.Error()})
			return
		}

		defer cancel()
		c.JSON(http.StatusOK, result)
	}
}

func itemByOrder(ctx context.Context, cancel context.CancelFunc, id string) (orderItems []primitive.M, err error) {
	matchStage, lookupStage, unwindStage := queryStage(id)
	lookupOrderStage, unwindOrderStage := queryOrderStage()
	lookupTableStage, unwindTableStage := queryTableStage()
	projectStage := queryProjectStage()
	groupStage := queryGroupStage()
	projectStage2 := queryProjectStage2()

	result, err := orderItemCollection.Aggregate(ctx, mongo.Pipeline{
		matchStage,
		lookupStage,
		unwindStage,
		lookupOrderStage,
		unwindOrderStage,
		lookupTableStage,
		unwindTableStage,
		projectStage,
		groupStage,
		projectStage2,
	})

	if err != nil {
		panic(err)
	}

	var OrderItems []primitive.M
	if err = result.All(ctx, &OrderItems); err != nil {
		panic(err)
	}

	defer cancel()

	return OrderItems, err
}

func queryStage(id string) (match, lookup, unwind bson.D) {
	/*
		- $match được sử dụng để lọc các tài liệu từ một bộ sưu tập dựa trên các điều kiện cho trước.
		- Key: "order_id", Value: id: đây là điều kiện để lọc các tài liệu trong bộ sưu tập.
			Trong trường hợp này, chúng ta muốn lọc các tài liệu mà có trường order_id có giá trị bằng id.

		=> câu lệnh này sẽ tạo ra một stage {$match} trong truy vấn aggregation,
			lọc các tài liệu trong bộ sưu tập sao cho trường order_id của chúng có giá trị bằng id.
	*/
	matchStage := bson.D{{Key: "$match", Value: bson.D{{Key: "order_id", Value: id}}}}
	/*
		- $lookup: Là một trong các stage của aggregation framework của MongoDB,
			được sử dụng để thực hiện việc join dữ liệu từ một bộ sưu tập (collection) khác vào trong
			bộ sưu tập hiện tại.
		- Key: "from", Value: "food": Đây là tên của bộ sưu tập mà chúng ta muốn tham gia vào truy vấn.
		- Key: "localField", Value: "food_id": Đây là trường trong bộ sưu tập hiện tại
			mà chúng ta sẽ sử dụng để so khớp với trường trong bộ sưu tập từ.
		- Key: "foreignField", Value: "food_id": Đây là trường trong bộ sưu tập từ mà chúng ta
			sẽ sử dụng để so khớp với trường trong bộ sưu tập hiện tại.
		- Key: "as", Value: "food": Đây là tên của trường mới mà chúng ta sẽ tạo ra
			sau khi thực hiện việc join.

		=> câu lệnh này sẽ tạo ra một stage {$lookup} trong truy vấn aggregation,
			thực hiện việc join dữ liệu từ bộ sưu tập "food" vào bộ sưu tập hiện tại
			dựa trên trường "food_id", và kết quả sẽ được lưu vào một trường mới có tên là "food".
	*/
	lookupStage := bson.D{
		{
			Key: "$lookup",
			Value: bson.D{
				{Key: "from", Value: "food"},
				{Key: "localField", Value: "food_id"},
				{Key: "foreignField", Value: "food_id"},
				{Key: "as", Value: "food"},
			},
		},
	}
	/*
		- $unwind: Là một trong các stage của aggregation framework của MongoDB,
			được sử dụng để tách các mảng (arrays) trong tài liệu thành các tài liệu riêng lẻ.
			Điều này hữu ích khi bạn muốn xử lý dữ liệu mảng như một tập hợp các tài liệu riêng lẻ.
		- Key: "path", Value: "$food": Đây là đường dẫn đến trường mảng trong tài liệu mà chúng ta muốn tách.
			Trong trường hợp này, chúng ta đang tách trường mảng "food".
		- Key: "preserveNullAndEmptyArrays", Value: true: Điều này xác định xem liệu các giá trị null
			hoặc mảng trống sẽ được bảo tồn trong kết quả sau khi tách hay không.
			Nếu được đặt là true, các giá trị null hoặc mảng trống sẽ được bảo tồn;
			nếu false, các tài liệu có chứa giá trị null hoặc mảng trống sẽ bị loại bỏ khỏi kết quả.

		=> câu lệnh này sẽ tạo ra một stage {$unwind} trong truy vấn aggregation,
			tách trường mảng "food" trong tài liệu thành các tài liệu riêng lẻ,
			và bảo tồn các giá trị null hoặc mảng trống trong kết quả sau khi tách.
	*/
	unwindStage := bson.D{
		{
			Key: "$unwind",
			Value: bson.D{
				{Key: "path", Value: "$food"},
				{Key: "preserveNullAndEmptyArrays", Value: true},
			},
		},
	}

	return matchStage, lookupStage, unwindStage
}

func queryOrderStage() (lookup, unwind bson.D) {
	/*
		- $lookup: Là một trong các stage của aggregation framework của MongoDB,
			được sử dụng để thực hiện việc join dữ liệu từ một bộ sưu tập (collection) khác vào trong
			bộ sưu tập hiện tại.
		- Key: "from", Value: "order": Đây là tên của bộ sưu tập mà chúng ta muốn tham gia vào truy vấn.
		- Key: "localField", Value: "order_id": Đây là trường trong bộ sưu tập hiện tại
			mà chúng ta sẽ sử dụng để so khớp với trường trong bộ sưu tập từ.
		- Key: "foreignField", Value: "order_id": Đây là trường trong bộ sưu tập từ mà chúng ta
			sẽ sử dụng để so khớp với trường trong bộ sưu tập hiện tại.
		- Key: "as", Value: "order": Đây là tên của trường mới mà chúng ta sẽ tạo ra
			sau khi thực hiện việc join.

		=> câu lệnh này sẽ tạo ra một stage {$lookup} trong truy vấn aggregation,
			thực hiện việc join dữ liệu từ bộ sưu tập "order" vào bộ sưu tập hiện tại
			dựa trên trường "order_id", và kết quả sẽ được lưu vào một trường mới có tên là "order".
	*/
	lookupOrderStage := bson.D{
		{
			Key: "$lookup",
			Value: bson.D{
				{Key: "from", Value: "order"},
				{Key: "localField", Value: "order_id"},
				{Key: "foreignField", Value: "order_id"},
				{Key: "as", Value: "order"},
			},
		},
	}
	/*
		- $unwind: Là một trong các stage của aggregation framework của MongoDB,
			được sử dụng để tách các mảng (arrays) trong tài liệu thành các tài liệu riêng lẻ.
			Điều này hữu ích khi bạn muốn xử lý dữ liệu mảng như một tập hợp các tài liệu riêng lẻ.
		- Key: "path", Value: "$order": Đây là đường dẫn đến trường mảng trong tài liệu mà chúng ta muốn tách.
			Trong trường hợp này, chúng ta đang tách trường mảng "order".
		- Key: "preserveNullAndEmptyArrays", Value: true: Điều này xác định xem liệu các giá trị null
			hoặc mảng trống sẽ được bảo tồn trong kết quả sau khi tách hay không.
			Nếu được đặt là true, các giá trị null hoặc mảng trống sẽ được bảo tồn;
			nếu false, các tài liệu có chứa giá trị null hoặc mảng trống sẽ bị loại bỏ khỏi kết quả.

		=> câu lệnh này sẽ tạo ra một stage {$unwind} trong truy vấn aggregation,
			tách trường mảng "order" trong tài liệu thành các tài liệu riêng lẻ,
			và bảo tồn các giá trị null hoặc mảng trống trong kết quả sau khi tách.
	*/
	unwindOrderStage := bson.D{
		{
			Key: "$unwind",
			Value: bson.D{
				{Key: "path", Value: "$order"},
				{Key: "preserveNullAndEmptyArrays", Value: true},
			},
		},
	}

	return lookupOrderStage, unwindOrderStage
}

func queryTableStage() (lookup, unwind bson.D) {
	/*
		- $lookup: Là một trong các stage của aggregation framework của MongoDB,
			được sử dụng để thực hiện việc join dữ liệu từ một bộ sưu tập (collection) khác vào trong
			bộ sưu tập hiện tại.
		- Key: "from", Value: "table": Đây là tên của bộ sưu tập mà chúng ta muốn tham gia vào truy vấn.
		- Key: "localField", Value: "order.table_id": Đây là trường trong bộ sưu tập hiện tại
			mà chúng ta sẽ sử dụng để so khớp với trường trong bộ sưu tập từ.
		- Key: "foreignField", Value: "table_id": Đây là trường trong bộ sưu tập từ mà chúng ta
			sẽ sử dụng để so khớp với trường trong bộ sưu tập hiện tại.
		- Key: "as", Value: "table": Đây là tên của trường mới mà chúng ta sẽ tạo ra
			sau khi thực hiện việc join.

		=> câu lệnh này sẽ tạo ra một stage {$lookup} trong truy vấn aggregation,
			thực hiện việc join dữ liệu từ bộ sưu tập "table" vào bộ sưu tập hiện tại
			dựa trên trường "table_id", và kết quả sẽ được lưu vào một trường mới có tên là "table".
	*/
	lookupTableStage := bson.D{
		{
			Key: "$lookup",
			Value: bson.D{
				{Key: "from", Value: "table"},
				{Key: "localField", Value: "order.table_id"},
				{Key: "foreignField", Value: "table_id"},
				{Key: "as", Value: "table"},
			},
		},
	}
	/*
		- $unwind: Là một trong các stage của aggregation framework của MongoDB,
			được sử dụng để tách các mảng (arrays) trong tài liệu thành các tài liệu riêng lẻ.
			Điều này hữu ích khi bạn muốn xử lý dữ liệu mảng như một tập hợp các tài liệu riêng lẻ.
		- Key: "path", Value: "$table": Đây là đường dẫn đến trường mảng trong tài liệu mà chúng ta muốn tách.
			Trong trường hợp này, chúng ta đang tách trường mảng "table".
		- Key: "preserveNullAndEmptyArrays", Value: true: Điều này xác định xem liệu các giá trị null
			hoặc mảng trống sẽ được bảo tồn trong kết quả sau khi tách hay không.
			Nếu được đặt là true, các giá trị null hoặc mảng trống sẽ được bảo tồn;
			nếu false, các tài liệu có chứa giá trị null hoặc mảng trống sẽ bị loại bỏ khỏi kết quả.

		=> câu lệnh này sẽ tạo ra một stage {$unwind} trong truy vấn aggregation,
			tách trường mảng "table" trong tài liệu thành các tài liệu riêng lẻ,
			và bảo tồn các giá trị null hoặc mảng trống trong kết quả sau khi tách.
	*/
	unwindTableStage := bson.D{
		{
			Key: "$unwind",
			Value: bson.D{
				{Key: "path", Value: "$table"},
				{Key: "preserveNullAndEmptyArrays", Value: true},
			},
		},
	}

	return lookupTableStage, unwindTableStage
}

func queryProjectStage() (project bson.D) {
	/*
		- $project: Là một trong các stage của aggregation framework của MongoDB,
			được sử dụng để chọn ra một hoặc nhiều trường từ tài liệu và chỉ định lại các tên trường
			hoặc tính toán trường mới.
		- Key: "id", Value: 0: Điều này xác định rằng trường "id" sẽ không xuất hiện trong kết quả cuối cùng.
		- Key: "amount", Value: "$food.price": Trường "amount" sẽ lấy giá trị của trường "price" từ tài liệu con "food".
		- Key: "total_count", Value: 1: Trường "total_count" sẽ được bảo tồn trong kết quả cuối cùng.
		- Key: "food_name", Value: "$food.name": Trường "food_name" sẽ lấy giá trị của trường "name" từ tài liệu con "food".
		- Key: "food_image", Value: "$food.food_image": Trường "food_image" sẽ lấy giá trị của trường "food_image" từ tài liệu con "food".
		- Key: "table_number", Value: "$table.table_number": Trường "table_number" sẽ lấy giá trị của trường "table_number" từ tài liệu con "table".
		- Key: "table_id", Value: "$table.table_id": Trường "table_id" sẽ lấy giá trị của trường "table_id" từ tài liệu con "table".
		- Key: "order_id", Value: "$order.order_id": Trường "order_id" sẽ lấy giá trị của trường "order_id" từ tài liệu con "order".
		- Key: "price", Value: "$food.price": Trường "price" sẽ lấy giá trị của trường "price" từ tài liệu con "food".
		- Key: "quantity", Value: 1: Trường "quantity" sẽ được thiết lập là 1 trong kết quả cuối cùng.

		=> câu lệnh này sẽ tạo ra một stage $project trong truy vấn aggregation,
			chọn ra các trường cần thiết từ các tài liệu con và chỉ định lại tên trường nếu cần.
	*/
	projectStage := bson.D{
		{
			Key: "$project",
			Value: bson.D{
				{Key: "id", Value: 0},
				{Key: "amount", Value: "$food.price"},
				{Key: "total_count", Value: 1},
				{Key: "food_name", Value: "$food.name"},
				{Key: "food_image", Value: "$food.food_image"},
				{Key: "table_number", Value: "$table.table_number"},
				{Key: "table_id", Value: "$table.table_id"},
				{Key: "order_id", Value: "$order.order_id"},
				{Key: "price", Value: "$food.price"},
				{Key: "quantity", Value: 1},
			},
		},
	}

	return projectStage
}

func queryGroupStage() (project bson.D) {
	//TODO: MISSING order_items
	/*
		- $group: Là một trong các stage của aggregation framework của MongoDB,
			được sử dụng để nhóm các tài liệu lại với nhau dựa trên các trường cụ thể và
			thực hiện các phép tính tổng hợp trên nhóm kết quả.
		- Key: "_id": Đây là trường đại diện cho các giá trị của các trường nhóm.
			Trong trường hợp này, chúng ta đang nhóm các tài liệu dựa trên các trường "order_id",
			"table_id" và "table_number".
		- Key: "payment_due", Value: bson.D{{Key: "$sum", Value: "$amount"}}:
			Đây là phép tính tổng hợp trên trường "amount" để tính tổng số tiền cần thanh toán
			("payment_due") trong từng nhóm.
		- Key: "total_count", Value: bson.D{{Key: "$sum", Value: 1}}:
			Đây là phép tính tổng hợp để đếm tổng số lượng tài liệu trong từng nhóm
			và lưu vào trường "total_count".
		- Key: "order_items", Value: bson.D{{Key: "$sum", Value: 1}}:
			Đây là phép tính tổng hợp để đếm tổng số lượng các mặt hàng đặt hàng trong từng nhóm
			và lưu vào trường "order_items".

		=> câu lệnh này sẽ tạo ra một stage $group trong truy vấn aggregation,
			nhóm các tài liệu dựa trên các trường "order_id", "table_id" và "table_number",
			và thực hiện các phép tính tổng hợp để tính tổng số tiền cần thanh toán,
			tổng số lượng tài liệu và tổng số lượng các mặt hàng đặt hàng trong từng nhóm.
	*/
	groupStage := bson.D{
		{
			Key: "$group",
			Value: bson.D{
				{
					Key: "_id",
					Value: bson.D{
						{Key: "order_id", Value: "$order_id"},
						{Key: "table_id", Value: "$table_id"},
						{Key: "table_number", Value: "$table_number"},
					},
				},
				{
					Key:   "payment_due",
					Value: bson.D{{Key: "$sum", Value: "$amount"}},
				},
				{
					Key:   "total_count",
					Value: bson.D{{Key: "$sum", Value: 1}},
				},
				{
					Key:   "order_items",
					Value: bson.D{{Key: "$sum", Value: 1}},
				},
			},
		},
	}

	return groupStage
}

func queryProjectStage2() (project bson.D) {
	/*
		- $project: Là một trong các stage của aggregation framework của MongoDB,
			được sử dụng để chọn ra một hoặc nhiều trường từ tài liệu
			và chỉ định lại các tên trường hoặc tính toán trường mới.
		- Key: "id", Value: 0: Điều này xác định rằng trường "id" sẽ không xuất hiện trong kết quả cuối cùng.
		- Key: "payment_due", Value: 1: Trường "payment_due" sẽ được bảo tồn trong kết quả cuối cùng.
		- Key: "total_count", Value: 1: Trường "total_count" sẽ được bảo tồn trong kết quả cuối cùng.
		- Key: "table_number", Value: "$_id.table_number":
			Trường "table_number" sẽ lấy giá trị của trường "table_number" từ trường "_id" của kết quả trước đó.
		- Key: "order_items", Value: 1: Trường "order_items" sẽ được bảo tồn trong kết quả cuối cùng.

		=> câu lệnh này sẽ tạo ra một stage $project trong truy vấn aggregation,
			chọn ra các trường cần thiết từ kết quả trước đó và chỉ định lại tên trường nếu cần.
			Trong trường hợp này, chúng ta cần lấy giá trị của trường "table_number" từ trường "_id"
			của kết quả trước đó.
	*/
	projectStage2 := bson.D{
		{
			Key: "$project",
			Value: bson.D{
				{Key: "id", Value: 0},
				{Key: "payment_due", Value: 1},
				{Key: "total_count", Value: 1},
				{Key: "table_number", Value: "$_id.table_number"},
				{Key: "order_items", Value: 1},
			},
		},
	}

	return projectStage2
}

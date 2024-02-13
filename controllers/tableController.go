package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/rongdo4897/restaurant-manager-go/database"
)

var tableCollection = database.OpenCollection(database.Client, "table")

func GetTables() gin.HandlerFunc {
	return func(c *gin.Context) {

	}
}

func GetTable() gin.HandlerFunc {
	return func(c *gin.Context) {

	}
}

func CreateTable() gin.HandlerFunc {
	return func(c *gin.Context) {

	}
}

func UpdateTable() gin.HandlerFunc {
	return func(c *gin.Context) {

	}
}

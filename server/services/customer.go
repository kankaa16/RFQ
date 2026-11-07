package services

import (
	"context"
	"net/http"
	"strings"
	"time"

	"rfq_costing/server/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type CustomerService struct {
	Collection *mongo.Collection
}

type CreateCustomerRequest struct {
	CompanyName       string `json:"company_name"`
	CustomerCode      string `json:"customer_code"`
	Industry          string `json:"industry"`
	CustomerType      string `json:"customer_type"`
	ContactPerson     string `json:"contact_person"`
	Email             string `json:"email"`
	Phone             string `json:"phone"`
	PaymentTerms      string `json:"payment_terms"`
	GSTIN             string `json:"gstin"`
	PreferredCurrency string `json:"preferred_currency"`
}

func (s *CustomerService) CreateCustomer(c *gin.Context) {
	var request CreateCustomerRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid request data",
		})
		return
	}

	request.CompanyName = strings.TrimSpace(request.CompanyName)
	request.CustomerCode = strings.ToUpper(strings.TrimSpace(request.CustomerCode))
	request.Email = strings.ToLower(strings.TrimSpace(request.Email))

	if request.CompanyName == "" || request.CustomerCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "company name and customer code are required",
		})
		return
	}

	if request.PreferredCurrency == "" {
		request.PreferredCurrency = "INR"
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	count, err := s.Collection.CountDocuments(ctx, bson.M{
		"customer_code": request.CustomerCode,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not check customer code",
		})
		return
	}

	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"message": "customer code already exists",
		})
		return
	}

	now := time.Now()

	customer := models.Customer{
		CompanyName:       request.CompanyName,
		CustomerCode:      request.CustomerCode,
		Industry:          request.Industry,
		CustomerType:      request.CustomerType,
		ContactPerson:     request.ContactPerson,
		Email:             request.Email,
		Phone:             request.Phone,
		PaymentTerms:      request.PaymentTerms,
		GSTIN:             request.GSTIN,
		PreferredCurrency: request.PreferredCurrency,
		IsActive:          true,
	}

	customer.CreatedAt = now
	customer.UpdatedAt = now

	result, err := s.Collection.InsertOne(ctx, customer)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not create customer",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "customer created successfully",
		"data": gin.H{
			"id":            result.InsertedID,
			"company_name":  customer.CompanyName,
			"customer_code": customer.CustomerCode,
		},
	})
}

func (s *CustomerService) ListCustomers(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	cursor, err := s.Collection.Find(ctx, bson.M{
		"is_active": true,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not fetch customers",
		})
		return
	}
	defer cursor.Close(ctx)

	var customers []models.Customer

	if err := cursor.All(ctx, &customers); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not read customers",
		})
		return
	}

	if customers == nil {
		customers = []models.Customer{}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"count":   len(customers),
		"data":    customers,
	})
}

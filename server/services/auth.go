package services

import (
	"context"
	"strings"
	"time"

	"rfq_costing/server/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	UserCollection *mongo.Collection
	JWTSecret      string
}

func (s *AuthService) Login(c *gin.Context) {
	var request map[string]string

	// parse login request
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, gin.H{
			"success": false,
			"message": "invalid request",
		})
		return
	}

	email := strings.ToLower(strings.TrimSpace(request["email"]))
	password := request["password"]

	// validate input
	if email == "" || password == "" {
		c.JSON(400, gin.H{
			"success": false,
			"message": "email and password are required",
		})
		return
	}

	// find user in mongodb
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var user models.User

	err := s.UserCollection.FindOne(ctx, bson.M{
		"email": email,
	}).Decode(&user)

	if err != nil {
		c.JSON(401, gin.H{
			"success": false,
			"message": "invalid email or password",
		})
		return
	}

	// verify password
	err = bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(password),
	)

	if err != nil {
		c.JSON(401, gin.H{
			"success": false,
			"message": "invalid email or password",
		})
		return
	}

	// generate jwt claims
	claims := jwt.MapClaims{
		"user_id": user.ID.Hex(),
		"email":   user.Email,
		"role":    user.Role,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}

	// create and sign jwt
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString([]byte(s.JWTSecret))

	if err != nil {
		c.JSON(500, gin.H{
			"success": false,
			"message": "could not generate token",
		})
		return
	}

	// return successful login response
	c.JSON(200, gin.H{
		"success": true,
		"message": "login successful",
		"data": gin.H{
			"token": signedToken,
			"user": gin.H{
				"id":    user.ID.Hex(),
				"name":  user.Name,
				"email": user.Email,
				"role":  user.Role,
			},
		},
	})
}

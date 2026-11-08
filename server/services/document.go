package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"rfq_costing/server/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type DocumentService struct {
	Collection *mongo.Collection
}

func (s *DocumentService) UploadDocument(c *gin.Context) {
	// rfq_id is optional for new rfq intake.
	var rfqID bson.ObjectID

	rfqIDValue := strings.TrimSpace(c.PostForm("rfq_id"))

	if rfqIDValue != "" {
		parsedID, err := bson.ObjectIDFromHex(rfqIDValue)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "invalid rfq id",
			})
			return
		}

		rfqID = parsedID
	}

	// validate the logged-in user.
	userID, err := bson.ObjectIDFromHex(c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "invalid logged-in user",
		})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "file is required",
		})
		return
	}

	extension := strings.ToLower(filepath.Ext(file.Filename))

	allowedExtensions := map[string]bool{
		".pdf":  true,
		".xlsx": true,
		".xls":  true,
		".csv":  true,
	}

	if !allowedExtensions[extension] {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "only pdf, xlsx, xls and csv files are allowed",
		})
		return
	}

	if file.Size <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "uploaded file is empty",
		})
		return
	}

	if file.Size > 20*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "file size must not exceed 20 mb",
		})
		return
	}

	openedFile, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not read uploaded file",
		})
		return
	}
	defer openedFile.Close()

	hash := sha256.New()

	if _, err := io.Copy(hash, openedFile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not process uploaded file",
		})
		return
	}

	fileHash := hex.EncodeToString(hash.Sum(nil))

	storedName := fmt.Sprintf(
		"%s%s",
		bson.NewObjectID().Hex(),
		extension,
	)

	uploadDirectory := filepath.Join("uploads", "rfq_documents")

	if err := os.MkdirAll(uploadDirectory, 0700); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not prepare upload directory",
		})
		return
	}

	storagePath := filepath.Join(uploadDirectory, storedName)

	if err := c.SaveUploadedFile(file, storagePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not save uploaded file",
		})
		return
	}

	category := strings.TrimSpace(c.PostForm("category"))
	if category == "" {
		category = models.DocumentTypePartList
	}

	document := models.Document{
		RFQID:            rfqID,
		FileName:         filepath.Base(file.Filename),
		StoredName:       storedName,
		FileType:         extension,
		FileSize:         file.Size,
		StoragePath:      storagePath,
		FileHash:         fileHash,
		Category:         category,
		UploadedBy:       userID,
		ExtractionStatus: models.ExtractionPending,
		IsVerified:       false,
	}

	now := time.Now()
	document.CreatedAt = now
	document.UpdatedAt = now

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	result, err := s.Collection.InsertOne(ctx, document)
	if err != nil {
		_ = os.Remove(storagePath)

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not save document details",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "document uploaded successfully",
		"data": gin.H{
			"id":        result.InsertedID,
			"file_name": document.FileName,
			"category":  document.Category,
		},
	})
}

package services

import (
	"context"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func normalizeHeader(value string) string {
	re := regexp.MustCompile(`[^a-z0-9]`)
	return re.ReplaceAllString(strings.ToLower(strings.TrimSpace(value)), "")
}

func parseNumber(value string) (float64, error) {
	value = strings.ReplaceAll(strings.TrimSpace(value), ",", "")
	return strconv.ParseFloat(value, 64)
}

func parseQuoteDate(value string) (time.Time, error) {
	layouts := []string{
		"2006-01-02",
		"02-01-2006",
		"02/01/2006",
		"2006/01/02",
	}

	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, strings.TrimSpace(value)); err == nil {
			return parsed, nil
		}
	}

	return time.Time{}, strconv.ErrSyntax
}

func (s *HistoricalQuoteService) ImportHistoricalQuotes(c *gin.Context) {
	userID, err := bson.ObjectIDFromHex(c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "invalid logged-in user",
		})
		return
	}

	header, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "excel file is required in the file field",
		})
		return
	}

	if header.Size == 0 || header.Size > 20*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "file must be between 1 byte and 20 mb",
		})
		return
	}

	if !strings.EqualFold(
		header.Filename[strings.LastIndex(header.Filename, ".")+1:],
		"xlsx",
	) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "only xlsx files are supported",
		})
		return
	}

	uploadedFile, err := header.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "could not open uploaded file",
		})
		return
	}
	defer uploadedFile.Close()

	workbook, err := excelize.OpenReader(uploadedFile)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid excel file",
		})
		return
	}
	defer workbook.Close()

	sheetName := ""

	for _, name := range workbook.GetSheetList() {
		if strings.EqualFold(name, "historical_quotes") {
			sheetName = name
			break
		}
	}

	if sheetName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "historical_quotes sheet not found",
		})
		return
	}

	rows, err := workbook.GetRows(sheetName)
	if err != nil || len(rows) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "excel sheet has no quote records",
		})
		return
	}

	columns := make(map[string]int)

	for index, value := range rows[0] {
		columns[normalizeHeader(value)] = index
	}

	requiredColumns := []string{
		"originalquotenumber",
		"customername",
		"partnumber",
		"partname",
		"materialgrade",
		"manufacturingprocess",
		"grossweightkg",
		"annualvolume",
		"materialcostinr",
		"conversioncostinr",
		"totalcostinr",
		"sellingpriceinr",
		"currency",
		"quotedat",
	}

	for _, column := range requiredColumns {
		if _, exists := columns[column]; !exists {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "missing excel column: " + column,
			})
			return
		}
	}

	getValue := func(row []string, column string) string {
		index := columns[column]

		if index >= len(row) {
			return ""
		}

		return strings.TrimSpace(row[index])
	}

	ctx, cancel := context.WithTimeout(
		c.Request.Context(),
		5*time.Minute,
	)
	defer cancel()

	imported := 0
	updated := 0
	skipped := 0
	errors := []string{}

	for rowIndex, row := range rows[1:] {
		excelRow := rowIndex + 2

		quoteNumber := getValue(row, "originalquotenumber")
		customerName := getValue(row, "customername")
		partNumber := getValue(row, "partnumber")
		partName := getValue(row, "partname")
		material := getValue(row, "materialgrade")
		process := getValue(row, "manufacturingprocess")
		currency := getValue(row, "currency")

		grossWeight, weightErr := parseNumber(
			getValue(row, "grossweightkg"),
		)
		annualVolume, volumeErr := parseNumber(
			getValue(row, "annualvolume"),
		)
		materialCost, materialErr := parseNumber(
			getValue(row, "materialcostinr"),
		)
		conversionCost, conversionErr := parseNumber(
			getValue(row, "conversioncostinr"),
		)
		totalCost, totalErr := parseNumber(
			getValue(row, "totalcostinr"),
		)
		sellingPrice, sellingErr := parseNumber(
			getValue(row, "sellingpriceinr"),
		)
		quotedAt, dateErr := parseQuoteDate(
			getValue(row, "quotedat"),
		)

		if quoteNumber == "" ||
			customerName == "" ||
			partNumber == "" ||
			partName == "" ||
			material == "" ||
			process == "" ||
			currency == "" ||
			weightErr != nil ||
			volumeErr != nil ||
			materialErr != nil ||
			conversionErr != nil ||
			totalErr != nil ||
			sellingErr != nil ||
			dateErr != nil ||
			grossWeight <= 0 ||
			annualVolume <= 0 ||
			annualVolume != float64(int64(annualVolume)) ||
			materialCost < 0 ||
			conversionCost < 0 ||
			totalCost < 0 ||
			sellingPrice < 0 {
			skipped++
			if len(errors) < 20 {
				errors = append(errors, "invalid data at row "+strconv.Itoa(excelRow))
			}
			continue
		}

		now := time.Now()

		document := bson.M{
			"original_quote_number": quoteNumber,
			"customer_name":         customerName,
			"part_number":           partNumber,
			"part_name":             partName,
			"material_grade":        material,
			"manufacturing_process": process,
			"gross_weight":          grossWeight,
			"annual_volume":         int64(annualVolume),
			"material_cost":         materialCost,
			"conversion_cost":       conversionCost,
			"total_cost":            totalCost,
			"selling_price":         sellingPrice,
			"currency":              currency,
			"quoted_at":             quotedAt,
			"source_file":           header.Filename,
			"imported_by":           userID,
			"updated_at":            now,
		}

		result, err := s.Collection.UpdateOne(
			ctx,
			bson.M{"original_quote_number": quoteNumber},
			bson.M{
				"$set": document,
				"$setOnInsert": bson.M{
					"created_at": now,
				},
			},
			options.UpdateOne().SetUpsert(true),
		)

		if err != nil {
			skipped++
			if len(errors) < 20 {
				errors = append(errors, "database error at row "+strconv.Itoa(excelRow))
			}
			continue
		}

		if result.UpsertedCount > 0 {
			imported++
		} else if result.ModifiedCount > 0 {
			updated++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"message":  "historical quote import completed",
		"imported": imported,
		"updated":  updated,
		"skipped":  skipped,
		"errors":   errors,
	})
}

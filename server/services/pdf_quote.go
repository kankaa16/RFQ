package services

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"rfq_costing/server/models"

	"github.com/gin-gonic/gin"
	"github.com/go-pdf/fpdf"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type PDFQuoteService struct {
	Collection *mongo.Collection
}

func (s *PDFQuoteService) GenerateQuotePDF(c *gin.Context) {
	quoteID, err := bson.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid quote id",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	var quote models.Quote

	err = s.Collection.FindOne(ctx, bson.M{
		"_id": quoteID,
	}).Decode(&quote)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "quote not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not fetch quote",
		})
		return
	}

	// only approved or sent quotes can be downloaded
	if quote.Status != models.QuoteStatusApproved &&
		quote.Status != models.QuoteStatusSent {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "only approved quotes can generate customer pdfs",
		})
		return
	}

	// create pdf
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// quotation heading
	pdf.SetFont("Arial", "B", 18)
	pdf.CellFormat(190, 12, "QUOTATION", "", 1, "C", false, 0, "")
	pdf.Ln(8)

	// quote details
	pdf.SetFont("Arial", "", 11)

	pdf.Cell(95, 8, "Quote Number: "+quote.QuoteNumber)
	pdf.Cell(95, 8, "Date: "+quote.CreatedAt.Format("02-01-2006"))
	pdf.Ln(10)

	pdf.Cell(190, 8, "Customer ID: "+quote.CustomerID.Hex())
	pdf.Ln(8)

	pdf.Cell(190, 8, "RFQ ID: "+quote.RFQID.Hex())
	pdf.Ln(14)

	// cost summary
	pdf.SetFont("Arial", "B", 11)

	pdf.CellFormat(95, 9, "Description", "1", 0, "L", false, 0, "")
	pdf.CellFormat(95, 9, "Amount (INR)", "1", 1, "R", false, 0, "")

	pdf.SetFont("Arial", "", 11)

	pdf.CellFormat(95, 9, "Subtotal", "1", 0, "L", false, 0, "")
	pdf.CellFormat(95, 9, formatAmount(quote.Subtotal), "1", 1, "R", false, 0, "")

	pdf.CellFormat(95, 9, "GST ("+strconv.FormatFloat(quote.GSTPercent, 'f', 2, 64)+"%)", "1", 0, "L", false, 0, "")
	pdf.CellFormat(95, 9, formatAmount(quote.GSTAmount), "1", 1, "R", false, 0, "")

	pdf.SetFont("Arial", "B", 11)

	pdf.CellFormat(95, 10, "Grand Total", "1", 0, "L", false, 0, "")
	pdf.CellFormat(95, 10, formatAmount(quote.GrandTotal), "1", 1, "R", false, 0, "")

	pdf.Ln(12)

	// terms and conditions
	pdf.SetFont("Arial", "", 10)

	pdf.MultiCell(190, 7,
		"Payment Terms: "+quote.PaymentTerms+
			"\nDelivery Terms: "+quote.DeliveryTerms+
			"\nDelivery Location: "+quote.DeliveryLocation+
			"\nValidity: "+strconv.Itoa(quote.ValidityDays)+" days",
		"", "L", false,
	)

	pdf.Ln(8)

	pdf.Cell(190, 8, "This quotation is subject to the terms mentioned above.")

	// save pdf
	uploadDirectory := filepath.Join("uploads", "quotes")

	if err := os.MkdirAll(uploadDirectory, 0700); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not prepare pdf directory",
		})
		return
	}

	fileName := quoteID.Hex() + ".pdf"
	pdfPath := filepath.Join(uploadDirectory, fileName)

	if err := pdf.OutputFileAndClose(pdfPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not generate quotation pdf",
		})
		return
	}

	// save pdf path in mongodb
	_, err = s.Collection.UpdateOne(ctx,
		bson.M{"_id": quoteID},
		bson.M{"$set": bson.M{
			"pdf_path":   pdfPath,
			"updated_at": time.Now(),
		}},
	)

	if err != nil {
		_ = os.Remove(pdfPath)

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not save pdf details",
		})
		return
	}

	// send pdf to the user
	c.FileAttachment(pdfPath, fileName)
}

func formatAmount(amount float64) string {
	return "Rs. " + strconv.FormatFloat(amount, 'f', 2, 64)
}

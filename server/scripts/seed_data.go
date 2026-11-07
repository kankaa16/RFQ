package main

import (
	"context"
	"log"
	"os"
	"time"

	"rfq_costing/server/models"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

type SamplePart struct {
	RFQNumber    string
	CustomerCode string
	CustomerName string
	Industry     string
	PartNumber   string
	PartName     string
	Material     string
	Process      string
	GrossWeight  float64
	AnnualVolume int64
	CycleTime    float64
	SetupTime    float64
	BatchSize    int64
	HourRate     float64
}

func upsertAndGetID(
	ctx context.Context,
	collection *mongo.Collection,
	filter bson.M,
	document interface{},
) (bson.ObjectID, error) {

	_, err := collection.ReplaceOne(
		ctx,
		filter,
		document,
		options.Replace().SetUpsert(true),
	)
	if err != nil {
		return bson.NilObjectID, err
	}

	var result struct {
		ID bson.ObjectID `bson:"_id"`
	}

	err = collection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		return bson.NilObjectID, err
	}

	return result.ID, nil
}

func main() {
	_ = godotenv.Load()

	uri := os.Getenv("MONGO_URI")
	databaseName := os.Getenv("MONGO_DATABASE")

	if uri == "" || databaseName == "" {
		log.Fatal("mongo configuration is missing")
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		60*time.Second,
	)
	defer cancel()

	client, err := mongo.Connect(
		options.Client().ApplyURI(uri),
	)
	if err != nil {
		log.Fatal("mongo connection failed:", err)
	}

	defer func() {
		_ = client.Disconnect(context.Background())
	}()

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		log.Fatal("mongo ping failed:", err)
	}

	db := client.Database(databaseName)

	userCollection := db.Collection("users")
	customerCollection := db.Collection("customers")
	rfqCollection := db.Collection("rfqs")
	partCollection := db.Collection("parts")
	rateCollection := db.Collection("rate_masters")
	routingCollection := db.Collection("routings")

	var estimator models.User

	err = userCollection.FindOne(ctx, bson.M{
		"email": os.Getenv("SEED_EMAIL"),
	}).Decode(&estimator)

	if err != nil {
		log.Fatal("estimator user not found; run seed_user.go first:", err)
	}

	now := time.Now()

	samples := []SamplePart{
		{
			RFQNumber:    "RFQ-2611-014",
			CustomerCode: "COMPANY-B",
			CustomerName: "Company B",
			Industry:     "two_wheeler",
			PartNumber:   "SPINDLE-001",
			PartName:     "Rear wheel spindle",
			Material:     "EN8",
			Process:      "machining",
			GrossWeight:  0.61,
			AnnualVolume: 180000,
			CycleTime:    3.2,
			SetupTime:    0,
			BatchSize:    500,
			HourRate:     850,
		},
		{
			RFQNumber:    "RFQ-2611-019",
			CustomerCode: "COMPANY-C",
			CustomerName: "Company C",
			Industry:     "tractor",
			PartNumber:   "LIFT-001",
			PartName:     "Hydraulic lift arm",
			Material:     "40Cr4 forging",
			Process:      "forging_and_machining",
			GrossWeight:  4.60,
			AnnualVolume: 24000,
			CycleTime:    11.5,
			SetupTime:    0,
			BatchSize:    500,
			HourRate:     1450,
		},
		{
			RFQNumber:    "RFQ-2611-023",
			CustomerCode: "COMPANY-D",
			CustomerName: "Company D",
			Industry:     "drivetrain",
			PartNumber:   "GEAR-001",
			PartName:     "Gear blank",
			Material:     "20MnCr5",
			Process:      "machining",
			GrossWeight:  1.48,
			AnnualVolume: 60000,
			CycleTime:    4.8,
			SetupTime:    0,
			BatchSize:    500,
			HourRate:     950,
		},
	}

	for _, sample := range samples {
		log.Println("seeding:", sample.RFQNumber)

		customer := models.Customer{
			BaseModel: models.BaseModel{
				CreatedAt: now,
				UpdatedAt: now,
			},
			CompanyName:       sample.CustomerName,
			CustomerCode:      sample.CustomerCode,
			Industry:          sample.Industry,
			CustomerType:      models.CustomerTypeOEM,
			PreferredCurrency: "INR",
			IsActive:          true,
		}

		customerID, err := upsertAndGetID(
			ctx,
			customerCollection,
			bson.M{"customer_code": sample.CustomerCode},
			customer,
		)
		if err != nil {
			log.Fatal("customer seeding failed:", err)
		}

		rfq := models.RFQ{
			BaseModel: models.BaseModel{
				CreatedAt: now,
				UpdatedAt: now,
			},
			RFQNumber:           sample.RFQNumber,
			CustomerID:          customerID,
			CustomerName:        sample.CustomerName,
			OwnerID:             estimator.ID,
			PartCount:           1,
			SourceChannel:       models.RFQSourceImport,
			Priority:            models.RFQPriorityMedium,
			ReceivedAt:          now,
			DueDate:             now.AddDate(0, 0, 7),
			ExpectedAnnualValue: 0,
			Currency:            "INR",
			Status:              models.RFQStatusEstimating,
		}

		rfqID, err := upsertAndGetID(
			ctx,
			rfqCollection,
			bson.M{"rfq_number": sample.RFQNumber},
			rfq,
		)
		if err != nil {
			log.Fatal("rfq seeding failed:", err)
		}

		part := models.Part{
			BaseModel: models.BaseModel{
				CreatedAt: now,
				UpdatedAt: now,
			},
			RFQID:                rfqID,
			PartNumber:           sample.PartNumber,
			PartName:             sample.PartName,
			MaterialGrade:        sample.Material,
			ManufacturingProcess: sample.Process,
			GrossWeight:          sample.GrossWeight,
			NetWeight:            sample.GrossWeight,
			ScrapWeight:          0,
			AnnualVolume:         sample.AnnualVolume,
			MonthlyVolume:        sample.AnnualVolume / 12,
			MaterialConfirmed:    true,
			DrawingVerified:      false,
		}

		partID, err := upsertAndGetID(
			ctx,
			partCollection,
			bson.M{
				"rfq_id":      rfqID,
				"part_number": sample.PartNumber,
			},
			part,
		)
		if err != nil {
			log.Fatal("part seeding failed:", err)
		}

		rateCode := "MACHINE-" + sample.RFQNumber

		rate := models.RateMaster{
			BaseModel: models.BaseModel{
				CreatedAt: now,
				UpdatedAt: now,
			},
			RateCode:      rateCode,
			RateName:      sample.PartName + " machine rate",
			RateType:      models.RateTypeMachine,
			Category:      "machining",
			MachineType:   "sample_machine",
			Rate:          sample.HourRate,
			Unit:          "INR/hour",
			Currency:      "INR",
			Version:       1,
			EffectiveFrom: now,
			IsActive:      true,
			UpdatedBy:     estimator.ID,
		}

		rateID, err := upsertAndGetID(
			ctx,
			rateCollection,
			bson.M{
				"rate_code": rateCode,
				"version":   1,
			},
			rate,
		)
		if err != nil {
			log.Fatal("rate seeding failed:", err)
		}

		routing := models.Routing{
			BaseModel: models.BaseModel{
				CreatedAt: now,
				UpdatedAt: now,
			},
			PartID: partID,
			Operations: []models.RoutingOperation{
				{
					Sequence:         1,
					Name:             "machining",
					MachineType:      "sample_machine",
					CycleTimeMinutes: sample.CycleTime,
					SetupTimeMinutes: sample.SetupTime,
					BatchSize:        sample.BatchSize,
					HourRate:         sample.HourRate,
					RateMasterID:     rateID,
				},
			},
			Source:     models.RoutingSourceManual,
			IsVerified: true,
			VerifiedBy: estimator.ID,
		}

		_, err = upsertAndGetID(
			ctx,
			routingCollection,
			bson.M{"part_id": partID},
			routing,
		)
		if err != nil {
			log.Fatal("routing seeding failed:", err)
		}

		log.Println("completed:", sample.RFQNumber)
	}

	log.Println("sample data seeded successfully")
}

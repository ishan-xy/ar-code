package handlers

import (
	"backend/config"
	"backend/database"
	"backend/utility"
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	utils "github.com/ItsMeSamey/go_utils"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var BucketName string = config.Cfg.BucketName
var ctx = context.Background()

type ModelReturnData struct {
	ID            primitive.ObjectID `json:"id"`
	DisplayName   string             `json:"display_name"`
	Query         string             `json:"query"`
	CreatedAt     time.Time          `json:"created_at"`
	FileExtension string             `json:"file_ext"`
	ModelURL      string             `json:"model_url"`
	QR_Code       string             `json:"qr_code"`
	Online        bool               `json:"online"`
	HasUSDZ       bool               `json:"has_usdz"` // true when a USDZ companion file is stored
}

type ModelUpdateData struct {
	DisplayName string `json:"display_name"`
	RefreshQR   bool   `json:"refresh_qr_code"`
	Online      bool   `json:"online"`
}

// uploadFileToS3 is a shared helper that streams a multipart file into S3 and
// returns the unique key that was used.
func uploadFileToS3(c fiber.Ctx, formKey, prefix, expectedExt string) (uniqueKey string, skipped bool, err error) {
	file, fErr := c.FormFile(formKey)
	if fErr != nil {
		// Field not present — caller decides whether that is an error.
		return "", true, nil
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != expectedExt {
		return "", false, fmt.Errorf("expected a %s file for field %q, got %q", expectedExt, formKey, ext)
	}

	f, fErr := file.Open()
	if fErr != nil {
		return "", false, utils.WithStack(fErr)
	}
	defer f.Close()

	if _, fErr = f.Seek(0, 0); fErr != nil {
		return "", false, utils.WithStack(fErr)
	}

	normalizedFilename := utility.NormalizeFileName(file.Filename)
	uniqueKey, fErr = utility.GenerateUniqueFilename(config.S3Client, BucketName, prefix, normalizedFilename)
	if fErr != nil {
		return "", false, utils.WithStack(fErr)
	}

	_, fErr = config.S3Client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: &BucketName,
		Key:    &uniqueKey,
		Body:   f,
	})
	if fErr != nil {
		return "", false, utils.WithStack(fErr)
	}

	return uniqueKey, false, nil
}

// deleteS3Keys removes one or more S3 keys, ignoring empty strings.
func deleteS3Keys(keys ...string) {
	var objs []types.ObjectIdentifier
	for _, k := range keys {
		if k != "" {
			k := k // capture
			objs = append(objs, types.ObjectIdentifier{Key: &k})
		}
	}
	if len(objs) == 0 {
		return
	}
	_, err := config.S3Client.DeleteObjects(context.Background(), &s3.DeleteObjectsInput{
		Bucket: &BucketName,
		Delete: &types.Delete{Objects: objs, Quiet: &[]bool{true}[0]},
	})
	if err != nil {
		log.Printf("S3 deleteS3Keys error: %v", err)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// GuestUploadModel — unauthenticated upload, GLB required, USDZ optional.
// POST /guest/model
// Form fields:
//   - model      (.glb, required)
//   - usdz       (.usdz, optional)
//   - displayName (string, optional)
//   - online     (true/false, optional, defaults to true)
// ──────────────────────────────────────────────────────────────────────────────

func GuestUploadModel(c fiber.Ctx) error {
	// --- GLB (required) ---
	glbKey, skipped, err := uploadFileToS3(c, "model", "guest", ".glb")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if skipped {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "model (.glb) file is required"})
	}

	// --- USDZ (optional) ---
	usdzKey, _, err := uploadFileToS3(c, "usdz", "guest", ".usdz")
	if err != nil {
		// USDZ upload failed — roll back the GLB we already uploaded.
		deleteS3Keys(glbKey)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	displayName := strings.TrimSpace(c.FormValue("displayName"))
	if displayName == "" {
		// derive from the GLB key
		base := filepath.Base(glbKey)
		displayName = strings.TrimSuffix(base, filepath.Ext(base))
	}

	expiresAt := time.Now().Add(72 * time.Hour)
	metadata := database.AR_model{
		ID:            primitive.NewObjectID(),
		OwnerID:       nil,
		FileName:      glbKey,
		USDZFileName:  usdzKey,
		DisplayName:   displayName,
		Query:         utility.GenerateQuery(glbKey),
		CreatedAt:     time.Now(),
		FileExtension: strings.TrimPrefix(filepath.Ext(glbKey), "."),
		Online:        true,
		IsGuest:       true,
		ExpiresAt:     &expiresAt,
	}

	if _, err = database.AR_modelDB.InsertOne(context.Background(), metadata); err != nil {
		deleteS3Keys(glbKey, usdzKey)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"query":      metadata.Query,
		"qr_code":    "/qr/" + metadata.Query,
		"ar_url":     "/model/files/" + metadata.Query,
		"has_usdz":   usdzKey != "",
		"expires_at": expiresAt,
	})
}

// CleanupExpiredGuestModels deletes guest models past their ExpiresAt time.
// Call this on startup and then on a ticker (e.g. every hour).
func CleanupExpiredGuestModels() {
	filter := bson.M{
		"is_guest":   true,
		"expires_at": bson.M{"$lt": time.Now()},
	}

	cursor, err := database.AR_modelDB.Collection.Find(context.Background(), filter)
	if err != nil {
		log.Printf("Cleanup: failed to find expired guests: %v", err)
		return
	}
	defer cursor.Close(context.Background())

	var expired []database.AR_model
	if err := cursor.All(context.Background(), &expired); err != nil || len(expired) == 0 {
		return
	}

	var objectIds []types.ObjectIdentifier
	var docIds []primitive.ObjectID
	for _, m := range expired {
		objectIds = append(objectIds, types.ObjectIdentifier{Key: &m.FileName})
		if m.USDZFileName != "" {
			objectIds = append(objectIds, types.ObjectIdentifier{Key: &m.USDZFileName})
		}
		docIds = append(docIds, m.ID)
	}

	_, err = config.S3Client.DeleteObjects(context.Background(), &s3.DeleteObjectsInput{
		Bucket: &BucketName,
		Delete: &types.Delete{Objects: objectIds, Quiet: &[]bool{true}[0]},
	})
	if err != nil {
		log.Printf("Cleanup: S3 batch delete error: %v", err)
	}

	res, err := database.AR_modelDB.DeleteMany(context.Background(), bson.M{"_id": bson.M{"$in": docIds}})
	if err != nil {
		log.Printf("Cleanup: MongoDB delete error: %v", err)
		return
	}
	log.Printf("Cleanup: removed %d expired guest model(s)", res.DeletedCount)
}

// ──────────────────────────────────────────────────────────────────────────────
// UploadModel — authenticated upload, GLB required, USDZ optional.
// POST /model
// Form fields:
//   - model      (.glb, required)
//   - usdz       (.usdz, optional)
//   - displayName (string, optional)
//   - online     (true/false, required)
// ──────────────────────────────────────────────────────────────────────────────

func UploadModel(c fiber.Ctx) error {
	userToken, _ := c.Locals("user").(*jwt.Token)
	_, username, err := utility.GetClaimsFromToken(userToken)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": utils.WithStack(err)})
	}
	user, _, err := database.UserDB.GetExists(bson.M{"username": username})
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": utils.WithStack(err)})
	}

	// --- GLB (required) ---
	glbKey, skipped, err := uploadFileToS3(c, "model", username, ".glb")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if skipped {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "model (.glb) file is required"})
	}

	// --- USDZ (optional) ---
	usdzKey, _, err := uploadFileToS3(c, "usdz", username, ".usdz")
	if err != nil {
		deleteS3Keys(glbKey)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	// --- Display name ---
	displayName := strings.TrimSpace(c.FormValue("displayName"))
	if displayName == "" {
		displayName = c.FormValue("display_name")
	}
	if displayName == "" {
		base := filepath.Base(glbKey)
		displayName = strings.TrimSuffix(base, filepath.Ext(base))
	}
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		deleteS3Keys(glbKey, usdzKey)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Display name cannot be empty"})
	}

	_, exists, err := database.AR_modelDB.GetExists(bson.M{"display_name": displayName, "owner_id": user.ID})
	if err != nil {
		deleteS3Keys(glbKey, usdzKey)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
	}
	if exists {
		deleteS3Keys(glbKey, usdzKey)
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "An object with the same name already exists"})
	}

	// --- Online flag ---
	var online bool
	switch c.FormValue("online") {
	case "true", "1":
		online = true
	case "false", "0":
		online = false
	default:
		deleteS3Keys(glbKey, usdzKey)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid publish value, must be true or false"})
	}

	metadata := database.AR_model{
		ID:            primitive.NewObjectID(),
		OwnerID:       &user.ID,
		FileName:      glbKey,
		USDZFileName:  usdzKey,
		DisplayName:   displayName,
		Query:         utility.GenerateQuery(glbKey),
		CreatedAt:     time.Now(),
		FileExtension: strings.TrimPrefix(filepath.Ext(glbKey), "."),
		Online:        online,
	}

	if _, err = database.AR_modelDB.InsertOne(context.Background(), metadata); err != nil {
		deleteS3Keys(glbKey, usdzKey)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":  "Model uploaded successfully",
		"metadata": metadata,
		"has_usdz": usdzKey != "",
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// GetModelRedirect — resolves a query to a presigned GLB URL.
// GET /model/files/:query
//
// If the User-Agent looks like Safari/iOS and a USDZ is available, redirect to
// the USDZ presigned URL instead so Quick Look activates on iPhone/iPad.
// ──────────────────────────────────────────────────────────────────────────────

func GetModelRedirect(c fiber.Ctx) error {
	query := c.Params("query")

	model, found, err := database.AR_modelDB.GetExists(bson.M{"query": query})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
	}
	if !found {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Model not found"})
	}
	if !model.Online {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Model is not online"})
	}

	// Prefer USDZ for Safari / WebKit (iOS Quick Look).
	ua := c.Get("User-Agent")
	wantUSDZ := model.USDZFileName != "" &&
		(strings.Contains(ua, "Safari") || strings.Contains(ua, "iPhone") || strings.Contains(ua, "iPad"))

	cacheKey := model.Query
	fileKey := model.FileName
	if wantUSDZ {
		cacheKey = model.Query + ":usdz"
		fileKey = model.USDZFileName
	}

	cachedURL, err := config.RedisClient.Get(ctx, cacheKey).Result()
	if err == redis.Nil {
		return generateAndCacheURL(c, fileKey, cacheKey)
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
	}
	if cachedURL == "" {
		return generateAndCacheURL(c, fileKey, cacheKey)
	}

	return c.Redirect().Status(fiber.StatusTemporaryRedirect).To(cachedURL)
}

func generateAndCacheURL(c fiber.Ctx, fileKey, cacheKey string) error {
	presignedURL, err := utility.GenerateR2PresignedURL(config.S3Client, BucketName, fileKey)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
	}

	if err := config.RedisClient.Set(ctx, cacheKey, presignedURL, 12*time.Hour).Err(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
	}

	return c.Redirect().Status(fiber.StatusTemporaryRedirect).To(presignedURL)
}

func GetModelMetadata(c fiber.Ctx) error {
	query := c.Params("query")

	model, found, err := database.AR_modelDB.GetExists(bson.M{"query": query})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
	}
	if !found {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Model not found"})
	}

	modelMeta := ModelReturnData{
		ID:            model.ID,
		DisplayName:   model.DisplayName,
		Query:         model.Query,
		CreatedAt:     model.CreatedAt,
		FileExtension: model.FileExtension,
		ModelURL:      "/model/files/" + model.Query,
		QR_Code:       "/qr/" + model.Query,
		Online:        model.Online,
		HasUSDZ:       model.USDZFileName != "",
	}

	return c.Status(fiber.StatusOK).JSON(modelMeta)
}

func GetAllModels(c fiber.Ctx) error {
	userToken := c.Locals("user").(*jwt.Token)
	claims := userToken.Claims.(jwt.MapClaims)
	user, _, err := database.UserDB.GetExists(bson.M{"username": claims["username"]})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
	}

	cursor, err := database.AR_modelDB.Collection.Find(context.Background(), bson.M{"owner_id": user.ID})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
	}
	defer cursor.Close(context.Background())

	var modelList []ModelReturnData
	for cursor.Next(context.Background()) {
		var model database.AR_model
		if err := cursor.Decode(&model); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
		}
		modelList = append(modelList, ModelReturnData{
			ID:            model.ID,
			DisplayName:   model.DisplayName,
			Query:         model.Query,
			CreatedAt:     model.CreatedAt,
			FileExtension: model.FileExtension,
			ModelURL:      "/model/files/" + model.Query,
			QR_Code:       "/qr/" + model.Query,
			Online:        model.Online,
			HasUSDZ:       model.USDZFileName != "",
		})
	}

	if err := cursor.Err(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
	}

	return c.Status(fiber.StatusOK).JSON(modelList)
}

func UpdateModel(c fiber.Ctx) error {
	var req ModelUpdateData
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   utils.WithStack(err),
			"message": "Invalid request body",
		})
	}

	query := c.Params("query")
	if query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Query parameter is required"})
	}

	model, found, err := database.AR_modelDB.GetExists(bson.M{"query": query})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
	}
	if !found {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Model not found"})
	}

	updateDoc := bson.M{}

	if req.DisplayName != "" && req.DisplayName != model.DisplayName {
		_, exists, err := database.AR_modelDB.GetExists(bson.M{
			"display_name": req.DisplayName,
			"owner_id":     model.OwnerID,
		})
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
		}
		if exists {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "A model with this display name already exists"})
		}
		updateDoc["display_name"] = req.DisplayName
		model.DisplayName = req.DisplayName
	}

	if req.RefreshQR {
		newQuery := utility.GenerateQuery(model.FileName)
		updateDoc["query"] = newQuery
		model.Query = newQuery
	}

	updateDoc["online"] = req.Online
	model.Online = req.Online

	if len(updateDoc) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "No fields to update"})
	}

	_, err = database.AR_modelDB.UpdateOne(context.Background(), bson.M{"_id": model.ID}, bson.M{"$set": updateDoc})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Model updated successfully",
		"model": ModelReturnData{
			ID:            model.ID,
			DisplayName:   model.DisplayName,
			Query:         model.Query,
			CreatedAt:     model.CreatedAt,
			FileExtension: model.FileExtension,
			ModelURL:      "/model/files/" + model.Query,
			QR_Code:       "/qr/" + model.Query,
			Online:        model.Online,
			HasUSDZ:       model.USDZFileName != "",
		},
	})
}

func DeleteModel(c fiber.Ctx) error {
	query := c.Params("query")
	if query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Query parameter is required"})
	}

	model, found, err := database.AR_modelDB.GetExists(bson.M{"query": query})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
	}
	if !found {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Model not found"})
	}

	// Delete both GLB and USDZ from S3.
	deleteS3Keys(model.FileName, model.USDZFileName)

	_, err = database.AR_modelDB.DeleteOne(context.Background(), bson.M{"_id": model.ID})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Model deleted successfully"})
}

type BulkDeleteRequest struct {
	Queries []string `json:"queries"`
}

func DeleteMultipleModels(c fiber.Ctx) error {
	userToken := c.Locals("user").(*jwt.Token)
	_, username, err := utility.GetClaimsFromToken(userToken)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": utils.WithStack(err)})
	}
	user, _, err := database.UserDB.GetExists(bson.M{"username": username})
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": utils.WithStack(err)})
	}

	queries := c.Query("query")
	if queries == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "At least one query parameter is required"})
	}
	queryList := strings.Split(queries, ",")
	if len(queryList) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "No valid queries provided"})
	}

	filter := bson.M{
		"query":    bson.M{"$in": queryList},
		"owner_id": user.ID,
	}
	cursor, err := database.AR_modelDB.Collection.Find(context.Background(), filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
	}
	defer cursor.Close(context.Background())

	var models []database.AR_model
	if err := cursor.All(context.Background(), &models); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
	}
	if len(models) == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "No models found for the provided queries"})
	}

	var objectIds []types.ObjectIdentifier
	modelIds := make([]primitive.ObjectID, 0, len(models))
	for _, model := range models {
		objectIds = append(objectIds, types.ObjectIdentifier{Key: &model.FileName})
		if model.USDZFileName != "" {
			objectIds = append(objectIds, types.ObjectIdentifier{Key: &model.USDZFileName})
		}
		modelIds = append(modelIds, model.ID)
	}

	_, err = config.S3Client.DeleteObjects(context.Background(), &s3.DeleteObjectsInput{
		Bucket: &BucketName,
		Delete: &types.Delete{
			Objects: objectIds,
			Quiet:   &[]bool{true}[0],
		},
	})
	if err != nil {
		log.Printf("Failed to delete some S3 objects: %v", err)
	}

	_, err = database.AR_modelDB.DeleteMany(context.Background(), bson.M{
		"_id": bson.M{"$in": modelIds},
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": fmt.Sprintf("Successfully deleted %d model(s)", len(models)),
	})
}

// GetARMeta returns minimal public metadata for the AR viewer.
func GetARMeta(c fiber.Ctx) error {
    model, found, err := database.AR_modelDB.GetExists(bson.M{"query": c.Params("query")})
    if err != nil   { return c.Status(500).JSON(fiber.Map{"error": err}) }
    if !found       { return c.Status(404).JSON(fiber.Map{"error": "not found"}) }
    if !model.Online { return c.Status(403).JSON(fiber.Map{"error": "offline"}) }
    return c.JSON(fiber.Map{"has_usdz": model.USDZFileName != ""})
}

// GetUSDZRedirect always serves the USDZ presigned URL (for iOS Quick Look).
func GetUSDZRedirect(c fiber.Ctx) error {
    model, found, err := database.AR_modelDB.GetExists(bson.M{"query": c.Params("query")})
    if err != nil || !found        { return c.Status(404).JSON(fiber.Map{"error": "not found"}) }
    if !model.Online               { return c.Status(403).JSON(fiber.Map{"error": "offline"}) }
    if model.USDZFileName == ""    { return c.Status(404).JSON(fiber.Map{"error": "no usdz"}) }
    return generateAndCacheURL(c, model.USDZFileName, model.Query+":usdz")
}
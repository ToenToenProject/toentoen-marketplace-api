package handlers

import (
	"net/http"
	"toentoen-backend/internal/models"
	"toentoen-backend/pkg/database"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func CreateProduct(c *fiber.Ctx) error {
	userToken := c.Locals("user").(*jwt.Token)
	claims := userToken.Claims.(jwt.MapClaims)
	userID := int64(claims["id"].(float64))
	role := claims["role"].(string)

	var input models.CreateProductInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid Input " + err.Error()})
	}

	if input.Name == "" || input.Price <= 0 || input.BusinessID == 0 {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Name, Price, and BusinessID are required fields and must be valid."})
	}

	if role == "owner" {
		var business models.Business
		if err := database.DB.Where("id = ? AND account_id = ?", input.BusinessID, userID).First(&business).Error; err != nil {
			return c.Status(http.StatusForbidden).JSON(fiber.Map{"error": "You do not have permission to add products to this business."})
		}
	}

	tx := database.DB.Begin()

	newProduct := models.Product{
		BusinessID:  input.BusinessID,
		Name:        input.Name,
		Description: input.Description,
		Price:       input.Price,
		StockStatus: input.StockStatus,
	}

	if newProduct.StockStatus == "" {
		newProduct.StockStatus = "in_stock"
	}

	for i, imgUrl := range input.Images {
		newProduct.Images = append(newProduct.Images, models.ProductImage{
			ImageURL:  imgUrl,
			SortOrder: i,
		})
	}

	if len(input.TagIDs) > 0 {
		var tags []models.Tag
		tx.Where("id IN ?", input.TagIDs).Find(&tags)
		newProduct.Tags = tags
	}

	if err := tx.Create(&newProduct).Error; err != nil {
		tx.Rollback()
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create product."})
	}

	tx.Commit()
	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"status":  "success",
		"message": "Produk berhasil ditambahkan",
		"data":    newProduct,
	})
}

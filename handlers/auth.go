package handlers

import (
	"backend/config"
	"backend/database"
	"backend/utility"
	"fmt"
	"log"

	utils "github.com/ItsMeSamey/go_utils"
	"github.com/gofiber/fiber/v3"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/v2/bson"
)



func SignUp(c fiber.Ctx) error {
	req := c.Locals("auth_request").(database.AuthRequest)
	
	_, exists, _ := database.UserDB.GetExists(bson.M{"email": req.Email})
	if exists {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Email already registered",})
	}

	hash, err := utility.HashPassword(req.Password)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   utils.WithStack(err),
			"message": "Failed to hash password",
		})
	}

	user := database.User{
		ID: primitive.NewObjectID(),
		Name: req.Name,
		Email:    req.Email,
		Password: hash,
	}
	_, err = database.UserDB.InsertOne(c.Context(), user)
	if err != nil {
		return utils.WithStack(err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message":"User Created"})
}

func Login(c fiber.Ctx) error {
	req := c.Locals("auth_request").(database.AuthRequest)

	user, exists, _ := database.UserDB.GetExists(bson.M{"email": req.Email})
	if !exists {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Email not registered"})
	}

	if !utility.CheckPasswordHash(req.Password, user.Password) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Incorrect password"})
	}

	tokenString, err := utility.GenerateJWT(user.Email)
	if err != nil {
		log.Println("Error generating JWT: ", err)
		return c.Status(fiber.StatusInternalServerError).JSON(utils.WithStack(err))
	}
	c.Set("Set-Cookie", fmt.Sprintf("%s=%s; HttpOnly; SameSite=Lax", config.Cfg.CookieName,tokenString))
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"token": fmt.Sprintf("Bearer %s", tokenString)})
}

func Logout(c fiber.Ctx) error {
	c.Set("Set-Cookie", fmt.Sprintf("%s=; HttpOnly; SameSite=Lax", config.Cfg.CookieName))
	return c.JSON(fiber.Map{"message": "Logout successful"})
}
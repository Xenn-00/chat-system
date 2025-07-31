package user_dto

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

type CreateUserRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6,max=100"`
}

type VerifyUserRequest struct {
	OTP string `json:"otp" validate:"required,otpval"`
}

var otpRegex = regexp.MustCompile(`^\d{6}$`)

func OTPValidator(fl validator.FieldLevel) bool {
	return otpRegex.MatchString(fl.Field().String())
}

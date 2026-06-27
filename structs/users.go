package structs

type UserCreate struct {
	Email string `json:"email" binding:"required,email"`
}

package domain

// RegisterRequest holds input payload for user registration.
type RegisterRequest struct {
	Username             string `json:"username"`
	Password             string `json:"password"`
	PasswordConfirmation string `json:"password_confirmation"`
}

// UserResponse holds output payload for user registration response.
type UserResponse struct {
	Username string `json:"username"`
}

// LoginRequest holds input payload for user login.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

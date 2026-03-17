package auth

type RegisterRequest struct {
	Email    string
	Name     string
	Password string
}

type LoginRequest struct {
	Email    string
	Password string
}

type RefreshRequest struct {
	RefreshTokenID string
	RefreshToken   string
}

type AuthResponse struct {
	AccessToken    string
	RefreshTokenID string
	RefreshToken   string
}

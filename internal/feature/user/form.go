package user

// Web form structs. They use `form` tags and the same validator as the API.

type LoginForm struct {
	Email    string `form:"email" validate:"required,email"`
	Password string `form:"password" validate:"required"`
	Next     string `form:"next"`
}

type SignupForm struct {
	Email    string `form:"email" validate:"required,email,max=254"`
	Name     string `form:"name" validate:"required,max=100"`
	Password string `form:"password" validate:"required,min=8,max=72"`
}

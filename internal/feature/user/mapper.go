package user

func toUserResponse(u User) UserResponse {
	return UserResponse{ID: u.ID, Email: u.Email, Name: u.Name, Role: u.Role, CreatedAt: u.CreatedAt.UTC()}
}

func (r CreateUserRequest) toInput() RegisterInput {
	return RegisterInput{Email: r.Email, Name: r.Name, Password: r.Password}
}

func (f SignupForm) toInput() RegisterInput {
	return RegisterInput{Email: f.Email, Name: f.Name, Password: f.Password}
}

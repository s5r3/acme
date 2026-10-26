package __name__

import (
	"time"

	"github.com/google/uuid"
)

type Create__Name__Request struct {
	Name string `json:"name" validate:"required,max=200"`
}

type List__Name__sQuery struct {
	Page     int `query:"page"`
	PageSize int `query:"pageSize" validate:"omitempty,max=100"`
}

type __Name__Response struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

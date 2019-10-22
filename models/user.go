package models

import (
	"encoding/json"

	"github.com/google/uuid"
)

// User Model
type User struct {
	ID              uuid.UUID
	Name            string
	Favorites       []*FoodTruck
	Reviews         []*Review
	Email           string
	OwnedFoodTrucks []*FoodTruck
}

type JSONUser struct {
	ID              string   `json:"id" bson:"_id"`
	Name            string   `json:"name" bson:"name"`
	Favorites       []string `json:"favorites" bson:"favorites"`
	Reviews         []string `json:"reviews" bson:"reviews"`
	Email           string   `json:"email" bson:"email"`
	OwnedFoodTrucks []string `json:"ownedFoodTrucks" bson:"ownedFoodTrucks"`
}

func NewJSONUser(user User) JSONUser {
	favorites := make([]string, len(user.Favorites))
	reviews := make([]string, len(user.Reviews))
	ownedFoodTrucks := make([]string, len(user.OwnedFoodTrucks))
	for i, favorite := range user.Favorites {
		favorites[i] = favorite.ID.String()
	}
	for i, review := range user.Reviews {
		reviews[i] = review.ID.String()
	}
	for i, ownedFoodTruck := range user.OwnedFoodTrucks {
		ownedFoodTrucks[i] = ownedFoodTruck.ID.String()
	}
	return JSONUser{
		ID:              user.ID.String(),
		Name:            user.Name,
		Favorites:       favorites,
		Reviews:         reviews,
		Email:           user.Email,
		OwnedFoodTrucks: ownedFoodTrucks,
	}
}

// MarshalJSON encodes a user into JSON
func (user User) MarshalJSON() ([]byte, error) {
	return json.Marshal(NewJSONUser(user))
}

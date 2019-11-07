package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"munchserver/middleware"
	"munchserver/models"
	"munchserver/queries"
	http "net/http"
	"regexp"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
)

type addFoodTruckRequest struct {
	Name        *string       `json:"name"`
	Address     *string       `json:"address"`
	Location    [2]float64    `json:"location"`
	Hours       *[7][2]string `json:"hours"`
	Photos      *[]string     `json:"photos"`
	Website     string        `json:"website"`
	PhoneNumber string        `json:"phoneNumber"`
	Description string        `json:"description"`
	Tags        []string      `json:"tags"`
}

type updateFoodTruckRequest struct {
	Name        string       `json:"name" bson:"name"`
	Address     string       `json:"address" bson:"address"`
	Location    [2]float64   `json:"location" bson:"location"`
	Owner       string       `json:"owner" bson:"owner"`
	Status      bool         `json:"status" bson:"status"`
	AvgRating   float32      `json:"avgRating" bson:"avgRating"`
	Hours       [7][2]string `json:"hours" bson:"hours"`
	Reviews     []string     `json:"reviews" bson:"reviews"`
	Photos      []string     `json:"photos" bson:"photos"`
	Website     string       `json:"website" bson:"website"`
	PhoneNumber string       `json:"phoneNumber" bson:"phoneNumber"`
	Description string       `json:"description" bson:"description"`
	Tags        []string     `json:"tags" bson:"tags"`
}

func PostFoodTrucksHandler(w http.ResponseWriter, r *http.Request) {
	// Get user from context
	user, userLoggedIn := r.Context().Value(middleware.UserKey).(string)

	// Check for a user, or if the user agent is from the scraper
	if !userLoggedIn && r.Header.Get("User-Agent") != "MunchCritic/1.0" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	foodTruckDecoder := json.NewDecoder(r.Body)
	foodTruckDecoder.DisallowUnknownFields()

	// Decode request
	var newFoodTruck addFoodTruckRequest
	err := foodTruckDecoder.Decode(&newFoodTruck)
	if err != nil {
		log.Printf("ERROR: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Make sure required fields are set
	if newFoodTruck.Name == nil ||
		newFoodTruck.Address == nil ||
		newFoodTruck.Hours == nil ||
		newFoodTruck.Photos == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Validate hours
	for i := 0; i < 7; i++ {
		validOpenTime, err := regexp.MatchString(`^\d{2}:\d{2}$`, newFoodTruck.Hours[i][0])
		if err != nil {
			log.Printf("ERROR: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		validCloseTime, err := regexp.MatchString(`^\d{2}:\d{2}$`, newFoodTruck.Hours[i][1])
		if err != nil {
			log.Printf("ERROR: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if !validOpenTime || !validCloseTime {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	// Generate uuid for food truck
	uuid, err := uuid.NewRandom()
	if err != nil {
		log.Printf("ERROR: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Set tags to an empty array if they don't exist
	tags := newFoodTruck.Tags
	if tags == nil {
		tags = []string{}
	}

	addedFoodTruck := models.JSONFoodTruck{
		ID:          uuid.String(),
		Name:        *newFoodTruck.Name,
		Address:     *newFoodTruck.Address,
		Location:    newFoodTruck.Location,
		Owner:       user,
		Hours:       *newFoodTruck.Hours,
		Reviews:     []string{},
		Photos:      *newFoodTruck.Photos,
		Website:     newFoodTruck.Website,
		PhoneNumber: newFoodTruck.PhoneNumber,
		Description: newFoodTruck.Description,
		Tags:        tags,
	}

	// Add food truck to database
	_, err = Db.Collection("foodTrucks").InsertOne(context.TODO(), addedFoodTruck)
	if err != nil {
		log.Printf("ERROR: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Update user that owns food truck
	if user != "" {
		_, err = Db.Collection("users").UpdateOne(context.TODO(), queries.WithID(user), queries.PushOwnedFoodTruck(uuid.String()))
		if err != nil {
			log.Printf("ERROR: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	// Send response
	w.WriteHeader(http.StatusOK)
}

func GetFoodTrucksHandler(w http.ResponseWriter, r *http.Request) {
	// Get all foodtrucks from the database into a cursor
	foodTrucksCollection := Db.Collection("foodTrucks")
	cur, err := foodTrucksCollection.Find(context.TODO(), bson.D{})
	if err != nil {
		log.Printf("ERROR: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Get users from cursor, convert to empty slice if no users in DB
	var foodTrucks []models.JSONFoodTruck
	cur.All(context.TODO(), &foodTrucks)
	if foodTrucks == nil {
		foodTrucks = make([]models.JSONFoodTruck, 0)
	}

	// Convert users to json
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(foodTrucks)
}

func PutFoodTrucksHandler(w http.ResponseWriter, r *http.Request) {

	// Checks for food truck ID
	params := mux.Vars(r)
	foodTruckID, foodTruckIDExists := params["foodTruckID"]
	if !foodTruckIDExists {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Get user from context
	user, userLoggedIn := r.Context().Value(middleware.UserKey).(string)

	fmt.Println(user)

	// Check for a user, or if the user agent is from the scraper
	if !userLoggedIn && r.Header.Get("User-Agent") != "MunchCritic/1.0" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	foodTruckDecoder := json.NewDecoder(r.Body)
	foodTruckDecoder.DisallowUnknownFields()

	// Decode request
	var currentFoodTruck updateFoodTruckRequest
	err := foodTruckDecoder.Decode(&currentFoodTruck)
	if err != nil {
		log.Printf("ERROR: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Set tags to an empty array if they don't exist
	tags := currentFoodTruck.Tags
	if tags == nil {
		tags = []string{}
	}

	// Determine which fields should be updated
	var updateData bson.D

	if currentFoodTruck.Name != "" {
		updateData = append(updateData, bson.E{"name", currentFoodTruck.Name})
	}
	if currentFoodTruck.Address != "" {
		updateData = append(updateData, bson.E{"address", currentFoodTruck.Address})
	}
	if len(currentFoodTruck.Location) != 0 {
		updateData = append(updateData, bson.E{"location", currentFoodTruck.Location})
	}
	if currentFoodTruck.Owner != "" {
		updateData = append(updateData, bson.E{"owner", currentFoodTruck.Owner})
	}
	if currentFoodTruck.Status != false {
		fmt.Println("changed status")
		updateData = append(updateData, bson.E{"status", currentFoodTruck.Status})
	}
	if currentFoodTruck.AvgRating != 0 {
		updateData = append(updateData, bson.E{"avgRating", currentFoodTruck.AvgRating})
	}
	// Validate hours if updating
	if currentFoodTruck.Hours[0][0] != "" {
		fmt.Println("---going into time")
		for i := 0; i < 7; i++ {
			validOpenTime, err := regexp.MatchString(`^\d{2}:\d{2}$`, currentFoodTruck.Hours[i][0])
			if err != nil {
				log.Printf("ERROR: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			validCloseTime, err := regexp.MatchString(`^\d{2}:\d{2}$`, currentFoodTruck.Hours[i][1])
			if err != nil {
				log.Printf("ERROR: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if !validOpenTime || !validCloseTime {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		}
	}
	if currentFoodTruck.Photos != nil {
		updateData = append(updateData, bson.E{"photos", currentFoodTruck.Photos})
	}
	if currentFoodTruck.Website != "" {
		updateData = append(updateData, bson.E{"website", currentFoodTruck.Website})
	}
	if currentFoodTruck.PhoneNumber != "" {
		updateData = append(updateData, bson.E{"phoneNumber", currentFoodTruck.PhoneNumber})
	}
	if currentFoodTruck.Description != "" {
		updateData = append(updateData, bson.E{"description", currentFoodTruck.Description})
	}
	if len(currentFoodTruck.Tags) != 0 {
		updateData = append(updateData, bson.E{"tags", tags})
	}

	// Update food truck document
	update := bson.D{
		{"$set", updateData},
	}

	_, err = Db.Collection("foodTrucks").UpdateOne(r.Context()), queries.WithID(foodTruckID), update)
	if err != nil {
		log.Fatal(err)
	}

	// Send response
	w.WriteHeader(http.StatusOK)

}

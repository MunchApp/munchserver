package routes

import (
	"encoding/json"
	"log"
	"munchserver/dbutils"
	"munchserver/middleware"
	"munchserver/models"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
)

type newReviewRequest struct {
	ReviewerName string    `json:"reviewerName"`
	FoodTruck    *string   `json:"foodTruck"`
	Comment      string    `json:"comment"`
	Rating       *float64  `json:"rating"`
	Date         time.Time `json:"date"`
	Origin       string    `json:"origin"`
}

func PostReviewsHandler(w http.ResponseWriter, r *http.Request) {
	// Get user from context
	user, userLoggedIn := r.Context().Value(middleware.UserKey).(string)

	// Check for a user, or if the user agent is from the scraper
	if !userLoggedIn && r.Header.Get("User-Agent") != "MunchCritic/1.0" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	reviewDecoder := json.NewDecoder(r.Body)
	reviewDecoder.DisallowUnknownFields()

	// Decode request
	var newReview newReviewRequest
	err := reviewDecoder.Decode(&newReview)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Make sure required fields set
	if (!userLoggedIn && newReview.ReviewerName == "") ||
		newReview.FoodTruck == nil ||
		newReview.Rating == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Lookup food truck
	var foodTruck models.JSONFoodTruck
	err = Db.Collection("foodTrucks").FindOne(r.Context(), dbutils.WithIDQuery(*newReview.FoodTruck)).Decode(&foodTruck)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Calculate rating sum to determine new avg rating, the new avg rating will be (that sum + review's rating) / new number of reviews
	ratingSum := float64(len(foodTruck.Reviews)) * foodTruck.AvgRating
	newAvgRating := (ratingSum + *newReview.Rating) / float64(len(foodTruck.Reviews)+1)

	// Generate uuid for review
	uuid, _ := uuid.NewRandom()

	date := newReview.Date
	if date.IsZero() {
		date = time.Now()
	}
	origin := newReview.Origin
	if origin == "" {
		origin = "munchapp"
	}
	reviewer := ""
	if userLoggedIn {
		reviewer = user
	}

	addedReview := models.JSONReview{
		ID:           uuid.String(),
		Reviewer:     reviewer,
		ReviewerName: newReview.ReviewerName,
		FoodTruck:    *newReview.FoodTruck,
		Comment:      newReview.Comment,
		Rating:       *newReview.Rating,
		Date:         date,
		Origin:       origin,
	}

	// Add review to database
	_, err = Db.Collection("reviews").InsertOne(r.Context(), addedReview)
	if err != nil {
		log.Printf("ERROR: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Attach review to user
	if userLoggedIn {
		_, err = Db.Collection("users").UpdateOne(r.Context(), dbutils.WithIDQuery(user), dbutils.PushReview(uuid.String()))
		if err != nil {
			log.Printf("ERROR: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	// Attach review to food truck
	_, err = Db.Collection("foodTrucks").UpdateOne(r.Context(), dbutils.WithIDQuery(*newReview.FoodTruck), dbutils.UpdateFoodTruckWithReview(newAvgRating, uuid.String()))
	if err != nil {
		log.Printf("ERROR: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(addedReview)
}

func GetReviewsOfFoodTruckHandler(w http.ResponseWriter, r *http.Request) {
	// Get food truck id from route params
	params := mux.Vars(r)
	foodTruckID, foodTruckIDExists := params["foodTruckID"]

	log.Printf("%v", params)

	if !foodTruckIDExists {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Check that food truck exists
	var foodTruck models.JSONFoodTruck
	foodTrucksCollection := Db.Collection("foodTrucks")
	err := foodTrucksCollection.FindOne(r.Context(), dbutils.WithIDQuery(foodTruckID)).Decode(&foodTruck)

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Get all reviews with foodtruck from the database into a cursor
	reviewsCollection := Db.Collection("reviews")
	cur, err := reviewsCollection.Find(r.Context(), dbutils.WithIDsQuery(foodTruck.Reviews))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("ERROR: %v", err)
		return
	}

	// Get reviews from cursor, convert to empty slice if no reviews in DB
	var reviews []models.JSONReview
	cur.All(r.Context(), &reviews)
	if reviews == nil {
		reviews = make([]models.JSONReview, 0)
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reviews)
}

func GetReviewsHandler(w http.ResponseWriter, r *http.Request) {
	// Get all reviews from the database into a cursor
	reviewsCollection := Db.Collection("reviews")
	cur, err := reviewsCollection.Find(r.Context(), bson.D{})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("ERROR: %v", err)
		return
	}

	// Get reviews from cursor, convert to empty slice if no reviews in DB
	var reviews []models.JSONReview
	cur.All(r.Context(), &reviews)
	if reviews == nil {
		reviews = make([]models.JSONReview, 0)
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reviews)
}

func GetReviewHandler(w http.ResponseWriter, r *http.Request) {

	// Get review ID from params
	params := mux.Vars(r)
	reviewID, reviewIDExists := params["reviewID"]

	if !reviewIDExists {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Get review from database
	var review models.JSONReview
	reviewsCollection := Db.Collection("reviews")
	err := reviewsCollection.FindOne(r.Context(), dbutils.WithIDQuery(reviewID)).Decode(&review)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("ERROR: %v", err)
		return
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(review)
}

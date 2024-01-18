package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type Receipt struct {
	Retailer     string `json:"retailer"`
	PurchaseDate string `json:"purchaseDate"`
	PurchaseTime string `json:"purchaseTime"`
	Items        []Item `json:"items"`
	Total        string `json:"total"`
}

type Item struct {
	ShortDescription string `json:"shortDescription"`
	Price            string `json:"price"`
}

type ReceiptResponse struct {
	ID string `json:"id"`
}

type ReceiptPoints struct {
	ID     string `json:"id"`
	Points int    `json:"points"`
}

type PointsScored struct {
	Points int `json:"points"`
}

// Source for storage since not using database
var pointsForReceipt []ReceiptPoints

func ProcessReceipt(w http.ResponseWriter, r *http.Request) {
	var receipt Receipt
	err := json.NewDecoder(r.Body).Decode(&receipt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	points := CalculatePoints(receipt)

	id := uuid.New().String()

	pointsForReceipt = append(pointsForReceipt, ReceiptPoints{ID: id, Points: points})

	response := ReceiptResponse{
		ID: id,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

}

func CalculatePoints(receipt Receipt) int {
	points := 0

	points += len(regexp.MustCompile("[a-zA-Z0-9]").FindAllString(receipt.Retailer, -1))

	totalFloat, err := strconv.ParseFloat(receipt.Total, 64)
	if err == nil && totalFloat == float64(int(totalFloat)) {
		points += 50
	}

	totalCents, err := strconv.ParseFloat(receipt.Total, 64)
	if err == nil {

		totalCentsInt := int(totalCents * 100)

		if totalCentsInt%25 == 0 {
			points += 25
		}
	}

	itemCount := len(receipt.Items)
	points += (itemCount / 2) * 5

	for _, item := range receipt.Items {
		trimmedLength := len(strings.TrimSpace(item.ShortDescription))
		price, err := strconv.ParseFloat(item.Price, 64)
		if err == nil && trimmedLength%3 == 0 {
			points += int(math.Ceil(price * 0.2))
		}
	}

	purchaseDate, err := time.Parse("2006-01-02", receipt.PurchaseDate)
	if err == nil && purchaseDate.Day()%2 == 1 {
		points += 6
	}

	purchaseTime, err := time.Parse("15:04", receipt.PurchaseTime)
	if err == nil && purchaseTime.After(time.Date(0, 1, 1, 14, 0, 0, 0, time.UTC)) && purchaseTime.Before(time.Date(0, 1, 1, 16, 0, 0, 0, time.UTC)) {
		points += 10
	}

	return points
}

func GetPointsForReceipt(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	receiptID := vars["id"]

	points := LookupPointsByReceiptID(receiptID)

	if points >= 0 {
		response := PointsScored{
			Points: points,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	} else {
		type ErrorResponse struct {
			Message string `json:"message"`
		}
		errorResponse := ErrorResponse{
			Message: fmt.Sprintf("Receipt with ID %s not found", receiptID),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errorResponse)
	}
}

func LookupPointsByReceiptID(receiptID string) int {
	for _, receipt := range pointsForReceipt {
		if receipt.ID == receiptID {
			return receipt.Points
		}
	}
	return -1
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/receipts/process", ProcessReceipt).Methods("POST")
	r.HandleFunc("/receipts/{id}/points", GetPointsForReceipt).Methods("GET")

	http.ListenAndServe(":8080", nil)

}

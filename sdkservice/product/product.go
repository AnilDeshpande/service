package product

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

const URL_ENDPOINT = "product"

type Response struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Cost        int       `json:"cost"`
	Quantity    int       `json:"quantity"`
	Sold        int       `json:"sold"`
	Revenue     int       `json:"revenue"`
	UserID      string    `json:"user_id"`
	DateCreated time.Time `json:"date_created"`
	DateUpdated time.Time `json:"date_updated"`
}

const product string = "http://localhost:3000/v1/products"

func GetProducts() error {

	request, reqErr := http.NewRequest("GET", product, nil)
	if reqErr != nil {
		return reqErr
	}

	// Create a Bearer string by appending string access token
	var bearer = "Bearer " + "eyJhbGciOiJSUzI1NiIsImtpZCI6IjU0YmIyMTY1LTcxZTEtNDFhNi1hZjNlLTdkYTRhMGUxZTJjMSIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJzdHVkZW50cyIsImV4cCI6MTY3OTYzNzUyNiwiaWF0IjoxNjc5NjMzOTI2LCJpc3MiOiJzZXJ2aWNlIHByb2plY3QiLCJzdWIiOiI1Y2YzNzI2Ni0zNDczLTQwMDYtOTg0Zi05MzI1MTIyNjc4YjciLCJyb2xlcyI6WyJBRE1JTiIsIlVTRVIiXX0.AJbwS4zXioXJn0fnVl39X8e3cKl8wt_qG-a-AG833ZBub-lJl1QC2HD0gSCqBFmKnZM93aPgR7ND-LcvAbvM6TOJrZFAPpZJRWmnCph9g8mfpcHd85Riu4kQPlUTv8xaZmjlU-89alzeOsVlXSDI6Huv9NjaNkKFb1qT26URfdU1qbRJRgUG5uAf93Ym7u6PcDoxePtihoDkv8bIa0ncHRXjhB7EQew21ceMMSdUDirFB9WszKIqhxSSdIqn8Goq6N_KHtm70mrCUXcBNPE-V80EDPC_4sxudXaA0dVKKmzaNr6jEsfArKHi1_nphAf7rbj4OD2MbMd9l2qitoZR6A"

	// add authorization header to the req
	request.Header.Add("Authorization", bearer)

	client := &http.Client{}

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode == 200 {

		var resData = []Response{}
		// Use json.Decode for reading streams of JSON data
		if err = json.NewDecoder(response.Body).Decode(&resData); err != nil {
			return err
		}
		fmt.Printf("Response: %+v\n", resData)
	} else {
		errdataBytes, _ := ioutil.ReadAll(response.Body)
		return fmt.Errorf("Got response code: %d, Error: %s", response.StatusCode, errdataBytes)
	}
	return nil
}

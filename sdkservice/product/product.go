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
	var bearer = "Bearer " + "eyJhbGciOiJSUzI1NiIsImtpZCI6IjU0YmIyMTY1LTcxZTEtNDFhNi1hZjNlLTdkYTRhMGUxZTJjMSIsInR5cCI6IkpXVCJ9.eyJhdWQiOiJzdHVkZW50cyIsImV4cCI6MTY3OTYwNjI2OCwiaWF0IjoxNjc5NjAyNjY4LCJpc3MiOiJzZXJ2aWNlIHByb2plY3QiLCJzdWIiOiI1Y2YzNzI2Ni0zNDczLTQwMDYtOTg0Zi05MzI1MTIyNjc4YjciLCJyb2xlcyI6WyJBRE1JTiIsIlVTRVIiXX0.j48ye_I4Jy0J_vDnB0YNmHETgiSFIn7YuLHfbTuUYCpTYUvPlgB7ceBiKYPLCjeET295aUgpqV7RcGMbm6bt_IBJM1Uvxm7rwBi7zMNXCUe_Aqsn6c3tjTj7cfpO1duJDfQ4eZCNChfhIIDQJDuItrpRkhGQ0WP9AHBT_vrayojTI7YIhxqeuVzmeTqVJUz0tvzBRGMJdF5SQ5Q7U_3SXynXMQWyWBOXYjaZhq6UxIgiMA9u29PJnrBrLOgmwMN1hUd9z3GUwrnFaJT6tOIBm0WM9VIDPvbl5v6tOKRvlo_gVC7V_b8tI0ZIQocYw2zJfcmfdbDmagESQcfTtAv4KA"

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

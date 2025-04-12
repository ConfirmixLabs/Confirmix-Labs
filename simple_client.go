package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

func main() {
	client := &http.Client{
		Timeout: time.Second * 5,
	}
	
	// İşlemleri listele
	getURL := "http://localhost:8080/api/transactions"
	fmt.Printf("Fetching transactions from %s\n", getURL)
	
	getResp, err := client.Get(getURL)
	if err != nil {
		fmt.Printf("Error fetching transactions: %v\n", err)
		return
	}
	defer getResp.Body.Close()
	
	getBody, err := ioutil.ReadAll(getResp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}
	
	fmt.Printf("Current transactions: %s\n\n", string(getBody))
	
	// Yeni oluşturduğumuz cüzdan ile bir işlem oluştur
	postURL := "http://localhost:8080/api/transaction"
	jsonData := `{
		"from": "f9f6318af7bfcdc5e7ead93544ca9baca30f9af255d2de78cf5ec794f97ec985",
		"to": "test-receiver-111",
		"value": 50.0
	}`
	
	fmt.Printf("Sending POST request to %s\n", postURL)
	req, _ := http.NewRequest("POST", postURL, strings.NewReader(jsonData))
	req.Header.Add("Content-Type", "application/json")
	
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	postBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}
	
	fmt.Printf("Transaction response: %s\n", string(postBody))
	fmt.Printf("Response status: %s\n", resp.Status)
	
	if resp.StatusCode == http.StatusCreated {
		fmt.Println("Transaction created successfully!")
		
		// İşlem oluştuktan sonra tekrar işlemleri listele
		fmt.Println("\nFetching transactions again to verify...")
		getResp2, err := client.Get(getURL)
		if err != nil {
			fmt.Printf("Error fetching transactions: %v\n", err)
			return
		}
		defer getResp2.Body.Close()
		
		getBody2, err := ioutil.ReadAll(getResp2.Body)
		if err != nil {
			fmt.Printf("Error reading response: %v\n", err)
			return
		}
		
		fmt.Printf("Updated transactions: %s\n", string(getBody2))
	} else {
		fmt.Println("Failed to create transaction.")
	}
} 
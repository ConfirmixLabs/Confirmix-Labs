package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

func main() {
	url := "http://localhost:8080/api/transactions"
	method := "GET"
	
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	
	fmt.Printf("Sending %s request to %s\n", method, url)
	
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}
	
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error sending request: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}
	
	fmt.Printf("Status: %s\n", resp.Status)
	fmt.Printf("Response Headers: %v\n", resp.Header)
	fmt.Printf("Response Body: %s\n", string(body))
	
	// Şimdi bir işlem oluşturalım
	createTxURL := "http://localhost:8080/api/transaction"
	createMethod := "POST"
	
	jsonData := `{
		"from": "test-sender-123",
		"to": "test-receiver-456",
		"value": 42
	}`
	
	fmt.Printf("\nSending %s request to %s\n", createMethod, createTxURL)
	
	createReq, err := http.NewRequest(createMethod, createTxURL, strings.NewReader(jsonData))
	if err != nil {
		fmt.Printf("Error creating transaction request: %v\n", err)
		return
	}
	
	createReq.Header.Add("Content-Type", "application/json")
	
	createResp, err := client.Do(createReq)
	if err != nil {
		fmt.Printf("Error sending transaction request: %v\n", err)
		return
	}
	defer createResp.Body.Close()
	
	createBody, err := ioutil.ReadAll(createResp.Body)
	if err != nil {
		fmt.Printf("Error reading transaction response: %v\n", err)
		return
	}
	
	fmt.Printf("Transaction Status: %s\n", createResp.Status)
	fmt.Printf("Transaction Response: %s\n", string(createBody))
	
	// Tekrar işlemleri sorgulayalım
	fmt.Printf("\nChecking transactions after creation...\n")
	
	checkReq, err := http.NewRequest(method, url, nil)
	if err != nil {
		fmt.Printf("Error creating check request: %v\n", err)
		return
	}
	
	checkResp, err := client.Do(checkReq)
	if err != nil {
		fmt.Printf("Error sending check request: %v\n", err)
		return
	}
	defer checkResp.Body.Close()
	
	checkBody, err := ioutil.ReadAll(checkResp.Body)
	if err != nil {
		fmt.Printf("Error reading check response: %v\n", err)
		return
	}
	
	fmt.Printf("Check Status: %s\n", checkResp.Status)
	fmt.Printf("Check Response: %s\n", string(checkBody))
} 
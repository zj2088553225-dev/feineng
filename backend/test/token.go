package main

import (
	"fmt"
	"github.com/anti-captcha/anticaptcha-go"
	"log"
)

func main() {
	// Create API client and set the API Key
	ac := anticaptcha.NewClient("10a08b8af36d99291ecab8281eff14e3")

	// set to 'false' to turn off debug output
	ac.IsVerbose = true

	// Specify softId to earn 10% commission with your app.
	// Get your softId here: https://anti-captcha.com/clients/tools/devcenter
	//ac.SoftId = 1187

	// Make sure the API key funds balance is positive
	balance, err := ac.GetBalance()
	if err != nil {
		log.Fatal(err)
		// Exit program to make sure you don't DDoS API with requests, while having empty balance
		return
	}
	fmt.Println("Balance:", balance)
	solution, err := ac.SolveImageFile("D:\\learn\\jumia\\backend\\test\\captcha.jpg", anticaptcha.ImageSettings{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Captcha Solution:", solution)
}

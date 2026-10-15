package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	method := os.Getenv("REQUEST_METHOD")
	path := os.Getenv("PATH_INFO")

	if method != "GET" || path != "/time" {
		fmt.Println(`{"error":"not found"}`)
		return
	}

	moscow := time.FixedZone("MSK", 3*60*60)
	now := time.Now().In(moscow)

	fmt.Printf(
		"{\"unix\":%d,\"iso\":%q,\"hour_minute\":%q}\n",
		now.Unix(),
		now.Format(time.RFC3339),
		now.Format("15:04"),
	)
}

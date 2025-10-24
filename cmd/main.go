package main

import (
	"log"
	"time"

	"github.com/zeljkobenovic/kmon/internal/app"
)

func main() {
	a, err := app.NewApp()
	if err != nil {
		log.Println("failed to instantiate pvman application: ", err.Error())
		time.Sleep(3 * time.Second)
	}

	if err = a.Run(); err != nil {
		log.Println("failed to run pvman application: ", err.Error())
		time.Sleep(3 * time.Second)
	}

	// allow timeout to see log output in K9s
	time.Sleep(3 * time.Second)
}

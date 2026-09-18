package main

import (
	"fmt"
	"github.com/tablehub/hub/internal/db"
)

func MainTestDB() {
	err := db.InitDB("./data/tablehub.db")
	if err != nil {
		fmt.Printf("InitDB error: %v\n", err)
		return
	}
	
	users, err := db.GetUsers()
	if err != nil {
		fmt.Printf("GetUsers error: %v\n", err)
		return
	}
	
	fmt.Printf("Users: %+v\n", users)
}

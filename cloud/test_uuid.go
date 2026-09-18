package main

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
)

type MyStruct struct {
	ID *uuid.UUID `json:"id"`
}

func main() {
	u := uuid.New()
	s := MyStruct{ID: &u}
	b, _ := json.Marshal(s)
	fmt.Println(string(b))
}

package main

import (
	"fmt"
	"time"
)

type Employee struct {
	ID        int
	Name      string
	Address   string
	DoB       time.Time
	Position  string
	Salary    int
	ManagerID int
}
var dilbert Employee

func EmployeeByID(id int) Employee {
	return Employee{
		ID:        id,
		Name:      "",
		Address:   "",
		DoB:       time.Time{},
		Position:  "武汉",
		Salary:    0,
		ManagerID: 0,
	}
}

func main(){
	fmt.Println(EmployeeByID(dilbert.ManagerID).Position) // "Pointy-haired boss"

	id := dilbert.ID
	A  := EmployeeByID(id)
	A.Salary = 1000 // fired for... no real reason
	fmt.Println(A)
}



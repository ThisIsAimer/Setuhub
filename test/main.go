package main

import (
	"fmt"
	"strconv"
)

func main() {

	number := "-2"

	num, _ := strconv.Atoi(number)

	fmt.Println(num)

	number = "-10"

	num, _ = strconv.Atoi(number)

	fmt.Println(num)
	number = "-100"

	num, _ = strconv.Atoi(number)

	fmt.Println(num)
	number = "-1000"

	num, _ = strconv.Atoi(number)

	fmt.Println(num)
	number = "-10000"

	num, _ = strconv.Atoi(number)

	fmt.Println(num)
	number = "-100000"

	num, _ = strconv.Atoi(number)

	fmt.Println(num)
	number = "-1000000"

	num, _ = strconv.Atoi(number)

	fmt.Println(num)

}

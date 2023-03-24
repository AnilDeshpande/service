package main

import (
	"fmt"

	"github.com/rahulgopher/sdkpoc/product"
)

func main() {
	if err := product.GetProducts(); err != nil {
		fmt.Print(err)
	}
}

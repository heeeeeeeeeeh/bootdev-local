package main

import (
	"fmt"
)

type divideError struct {
	dividend float64
}

func (u divideError) Error() string {
	return fmt.Sprintf("can not divide %v by zero", u.dividend)
}

func divide(dividend, divisor float64) (float64, error) {
	if divisor == 0 {
		return 0, divideError{dividend: dividend}
	}
	return dividend / divisor, nil
}

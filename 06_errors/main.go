package main

/*
TASK 6: Error Trees, Wrapping, Custom Error Types, and errors.Is / errors.As

Topic: Advanced Error Handling – Error Trees & Custom Types

Problem Description:
You are building an Error Handling & Diagnostic Subsystem for an E-Commerce / Payment Service.
In production Go, errors form a rich hierarchy/chain across application layers (Repository -> Service -> Transport/API).
Your system must distinguish between operational sentinel errors, structured validation errors,
and transport/API errors, allowing upstream callers to inspect, extract metadata, and unwrap error chains cleanly.

Requirements:

1. Sentinel Errors:
   - Define sentinel errors using `errors.New`:
     * `ErrNotFound`: "record not found"
     * `ErrPermissionDenied`: "permission denied"
     * `ErrNetworkTimeout`: "network timeout"

2. Custom Structured Error Types:
   - `ValidationError` struct (fields: `Field string`, `Reason string`):
     * Implements `Error() string` (e.g., "validation failed on field '<Field>': <Reason>").
   - `APIError` struct (fields: `HTTPCode int`, `Message string`, `Err error`):
     * Implements `Error() string`.
     * Implements `Unwrap() error`: returns the underlying wrapped error `Err` so `errors.Is` and `errors.As` can traverse the chain.

3. Field Validator using `errors.Join` (Go 1.20+):
   - `User` struct (fields: `Username string`, `Email string`, `Age int`).
   - Implement `ValidateUser(u User) error`:
     * Validates:
       1) `Username` cannot be empty.
       2) `Email` must contain "@".
       3) `Age` must be >= 18.
     * For each invalid rule, create a `*ValidationError`.
     * If multiple rules fail, combine all validation errors using `errors.Join(errs...)` and return the joined error.
     * If valid, return `nil`.

4. Layered Service Simulation with Error Wrapping (`fmt.Errorf` with `%w`):
   - Implement `GetUser(userID int) (*User, error)`:
     * If `userID == 0`: returns `nil` and an error wrapped with `ErrNotFound` using `%w` (e.g., `fmt.Errorf("repo query failed: %w", ErrNotFound)`).
     * If `userID < 0`: returns `nil` and an error wrapped with `ErrNetworkTimeout` using `%w`.
     * Otherwise returns a valid `User` and `nil`.
   - Implement `HandlePaymentRequest(userID int, amount float64) error`:
     * Calls `GetUser(userID)`.
     * If fetching fails, wraps the error inside an `*APIError` (with appropriate `HTTPCode`, e.g. 404 for not found, 503 for timeout) and returns it.
     * Validates user with `ValidateUser`. If invalid, wraps inside `*APIError` with HTTPCode 400.
     * If amount <= 0, returns an `*APIError` with HTTPCode 422.

5. In `main()` function:
   - Demonstrate inspecting errors using:
     a) `errors.Is(err, ErrNotFound)` and `errors.Is(err, ErrNetworkTimeout)` on an `APIError`.
     b) `errors.As(err, &apiErr)` to extract the `HTTPCode` and message from the error chain.
     c) `errors.As(err, &valErr)` or checking joined validation errors.
   - Run and verify with `go run 06_errors/main.go`.

Good luck! Implement your solution below.
*/

import (
	"errors"
	"fmt"
	"strings"
)

// TODO 1: Define sentinel errors (ErrNotFound, ErrPermissionDenied, ErrNetworkTimeout)
var (
	ErrNotFound         = errors.New("record not found")
	ErrPermissionDenied = errors.New("permission denied")
	ErrNetworkTimeout   = errors.New("network timeout")
)

type ValidationError struct {
	Field  string
	Reason string
}

func (v ValidationError) Error() string {
	return fmt.Sprintf("validation failed on field '%s': '%s'", v.Field, v.Reason)
}

type ValidationErrors []ValidationError

func (ve ValidationErrors) Error() string {
	strs := make([]string, len(ve))
	for i := range ve {
		strs[i] = ve[i].Error()
	}

	return strings.Join(strs, "\n")
}

type APIError struct {
	HTTPCode int
	Message  string
	Err      error
}

func (v APIError) Error() string {
	return fmt.Sprintf("api error; code: %d, message %s", v.HTTPCode, v.Message)
}

func (v APIError) Unwrap() error {
	return v.Err
}

type User struct {
	Username string
	Email    string
	Age      int
}

func ValidateUser(u User) ValidationErrors {
	errs := make(ValidationErrors, 0)
	if u.Username == "" {
		errs = append(errs, ValidationError{Field: "Username", Reason: "`Username` cannot be empty"})
	}
	if !strings.Contains(u.Email, "@") {
		errs = append(errs, ValidationError{Field: "Email", Reason: "`Email` must contain \"@\""})
	}
	if u.Age < 18 {
		errs = append(errs, ValidationError{Field: "Age", Reason: "`Age` must be >= 18"})
	}

	if len(errs) == 0 {
		return nil
	}

	return errs
}

func GetUser(userID int) (*User, error) {
	if userID == 0 {
		return nil, fmt.Errorf("repo query failed: %w", ErrNotFound)
	}
	if userID < 0 {
		return nil, fmt.Errorf("repo timeout: %w", ErrNetworkTimeout)
	}
	var email string
	if userID%2 == 0 {
		email = fmt.Sprintf("user_%d@go.local", userID)
	} else {
		email = fmt.Sprintf("user_%d#go.local", userID)
	}

	return &User{
		Username: fmt.Sprintf("user_%d", userID),
		Email:    email,
		Age:      userID,
	}, nil
}

func HandlePaymentRequest(userID int, amount float64) error {
	if amount <= 0 {
		return &APIError{
			HTTPCode: 422,
			Message:  "invalid amount",
		}
	}

	user, err := GetUser(userID)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			return &APIError{
				HTTPCode: 404,
				Message:  "payment request failed because user is not found",
				Err:      err,
			}
		case errors.Is(err, ErrNetworkTimeout):
			return &APIError{
				HTTPCode: 504,
				Message:  "payment request due to a timeout",
				Err:      err,
			}
		default:
			return &APIError{
				HTTPCode: 500,
				Message:  "internal server error",
				Err:      err,
			}
		}
	}

	validationErrors := ValidateUser(*user)
	if validationErrors != nil {
		return &APIError{
			HTTPCode: 400,
			Message:  "user validation error",
			Err:      validationErrors,
		}
	}

	return nil
}

func main() {
	fmt.Println("Task 6: Error Trees, Wrapping, and errors.Is / errors.As\n")

	fmt.Println("Valid user 20, age 20, all good, no errors")
	printErrInfo(HandlePaymentRequest(20, 100.))

	fmt.Println("\nUser not found")
	printErrInfo(HandlePaymentRequest(0, 100.))

	fmt.Println("\nNetwork error")
	printErrInfo(HandlePaymentRequest(-10, 100.))

	fmt.Println("\nUser validation errors")
	printErrInfo(HandlePaymentRequest(11, 100.))

	fmt.Println("\nAmount error")
	printErrInfo(HandlePaymentRequest(20, 0.0))
}

func printErrInfo(err error) {
	if err == nil {
		fmt.Println("No errors")
		return
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) {
		fmt.Printf("HTTP Status: %d, Message: %s\n", apiErr.HTTPCode, apiErr.Message)
	}

	var valErrors ValidationErrors
	if errors.As(err, &valErrors) {
		for _, v := range valErrors {
			fmt.Printf(" - %s: %s\n", v.Field, v.Reason)
		}
	}

	// alternative
	// if errors.As(err, &joinedErrors) {
	// 	for _, e := range joinedErrors.Unwrap() {
	// 		var validationErr *ValidationError
	// 		if errors.As(e, &validationErr) {
	// 			fmt.Printf(" - validation: %s\n", validationErr.Error())
	// 		}
	// 	}
	// }
}

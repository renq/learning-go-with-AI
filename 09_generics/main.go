package main

/*
TASK 9: Generic Functions, Type Constraints, and Slice Transformations

Topic: Generics & Type System – Type Parameters & Constraints

Problem Description:
You are building a high-performance, reusable Functional Utility Library in Go.
Prior to Generics (Go 1.18+), developers had to duplicate slice utilities for every data type or rely
on `any` / `interface{}` with reflection, which introduced heap allocations and runtime type crashes.
You will implement a type-safe generic toolkit using type parameters, custom constraint unions,
type approximation (`~T`), and the `comparable` constraint.

Requirements:

1. Custom Type Constraints:
   - Define a constraint `Number` supporting integer and float types:
     (signed integers, unsigned integers, and floats).
     * IMPORTANT: Use type approximation (tilde `~`, e.g., `~int`, `~float64`) so custom defined types
       (e.g., `type UserID int`, `type USD float64`) satisfy the constraint.
   - Define a constraint `Ordered` supporting all ordered types (all `Number` types plus `~string`).

2. Generic Slice Transformations:
   - Implement `Map[T any, R any](items []T, fn func(T) R) []R`:
     * Transforms every element of type `T` into type `R`.
   - Implement `Filter[T any](items []T, predicate func(T) bool) []T`:
     * Returns a new slice containing only elements for which `predicate(item)` returns `true`.
   - Implement `Reduce[T any, R any](items []T, initial R, fn func(acc R, item T) R) R`:
     * Aggregates elements of type `T` into a single value of type `R` using an accumulator function.

3. Generic Deduplication & Search:
   - Implement `Unique[T comparable](items []T) []T`:
     * Deduplicates elements while preserving the original order of their first appearance.
     * Must use the built-in `comparable` constraint.
   - Implement `Min[T Ordered](items []T) (T, error)` and `Max[T Ordered](items []T) (T, error)`:
     * Finds the minimum / maximum element using `<` or `>` comparisons.
     * If the slice is empty, returns the zero value of `T` and an error (`errors.New("slice is empty")`).

4. In `main()` function:
   - Demonstrate:
     a) Using custom types with type approximation (e.g., `type Price float64`) with `Min`, `Max`, and `Reduce`.
     b) Transforming a slice of custom structs (e.g., `Product{Name string, Price Price}`) into a slice of strings using `Map`.
     c) Filtering items with `Filter`.
     d) Deduplicating items with `Unique`.
     e) Calling `Min` on an empty slice and handling the returned error.
   - Run and verify with `go run 09_generics/main.go`.

Good luck! Implement your solution below.
*/

import (
	"errors"
	"fmt"
)

type Number interface {
	~int | ~float64
}

type Ordered interface {
	Number | ~string
}

func Map[T any, R any](items []T, fn func(T) R) []R {
	result := make([]R, len(items))
	for i := range items {
		result[i] = fn(items[i])
	}

	return result
}

func Filter[T any](items []T, fn func(T) bool) []T {
	result := make([]T, 0)
	for i := range items {
		if fn(items[i]) {
			result = append(result, items[i])
		}
	}
	return result
}

func Reduce[T any, R any](items []T, initial R, fn func(acc R, item T) R) R {
	result := initial
	for i := range items {
		result = fn(result, items[i])
	}
	return result
}

func Unique[T comparable](items []T) []T {
	seen := make(map[T]struct{}, len(items)) // comparable can be a key
	result := make([]T, 0, len(items))

	for _, item := range items {
		if _, exists := seen[item]; !exists {
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}

var EmptySlice = errors.New("slice is empty")

func Min[T Ordered](items []T) (T, error) {
	if len(items) == 0 {
		var t T
		return t, EmptySlice
	}

	if len(items) == 1 {
		return items[0], nil
	}

	min := items[0]
	for i := 1; i < len(items); i++ {
		if items[i] < min {
			min = items[i]
		}
	}

	return min, nil
}

func Max[T Ordered](items []T) (T, error) {
	if len(items) == 0 {
		var t T
		return t, EmptySlice
	}

	if len(items) == 1 {
		return items[0], nil
	}

	max := items[0]
	for i := 1; i < len(items); i++ {
		if max < items[i] {
			max = items[i]
		}
	}

	return max, nil
}

func Must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

type Price float64

type Product struct {
	Name  string
	Price Price
}

func printProducts(products []Product) {
	fmt.Print(Reduce(
		products,
		"",
		func(acc string, product Product) string {
			return acc + fmt.Sprintf("  %s %.2f\n", product.Name, product.Price)
		},
	))
	fmt.Println()
}

func main() {
	fmt.Println("Task 9: Generic Functions & Type Constraints")

	products := []Product{
		{
			Name:  "Apples",
			Price: 8.0,
		},
		{
			Name:  "Plums",
			Price: 14.0,
		},
		{
			Name:  "Watermelon",
			Price: 15.0,
		},
		{
			Name:  "Avocado",
			Price: 39.9,
		},
		{
			Name:  "Onion",
			Price: 5.90,
		},
		{
			Name:  "Garlic",
			Price: 5.90,
		},
	}

	fmt.Println("Products:")
	printProducts(products)

	prices := Map(products, func(p Product) Price {
		return p.Price
	})

	fmt.Printf("Lowest price: %.2f\n", Must(Min(prices)))
	fmt.Printf("Highest price: %.2f\n\n", Must(Max(prices)))

	fmt.Println("Cheap products < $10:")
	printProducts(Filter(products, func(p Product) bool {
		return p.Price < 10
	}))

	fmt.Printf("Unique prices: %s\n", Reduce(
		Unique(prices),
		"",
		func(acc string, p Price) string {
			if acc == "" {
				return fmt.Sprintf("%.2f", p)
			}
			return fmt.Sprintf("%s, %.2f", acc, p)
		},
	))
}

package main

import (
	"fmt"
	"errors"
	"strings"

	"github.com/goldchell-bra7/gos1mple/input"
	"github.com/goldchell-bra7/gos1mple/text"
)

var err error
var result, first, second float64
var operator string

func plus(first float64, second float64) float64 {
	result := first + second
	return result
}

func minus(first float64, second float64) float64 {
        result := first - second
        return result
}

func multiply(first float64, second float64) float64 {
        result := first * second
        return result
}

func divide(first float64, second float64) (float64, error) {
	if second == 0 {
		return 0, errors.New("На 0 делить нельзя!")
	} else {
	 result := first / second
        return result, nil
	}
}

func main() {
	MainLoop:
	for {

		fmt.Println(text.Header("Калькулятор", "=", 40))
		fmt.Println("Выберите операцию по номеру")
		fmt.Println("1. Сложение")
		fmt.Println("2. Вычитание")
		fmt.Println("3. Умножение")
		fmt.Println("4. Деление")
		fmt.Println("5. Выход")
		fmt.Println("Номер операции: ")
		fmt.Scan(&operator)

		switch operator {

			case "1":
				first = input.ReadFloat("Введи первое число: ")
				second = input.ReadFloat("Введи второе число: ")
				result = plus(first, second)
				fmt.Println("Ваш результат: ", result)

			case "2":
				first = input.ReadFloat("Введи первое число: ")
				second = input.ReadFloat("Введи второе число: ")
				result = minus(first, second)
				fmt.Println("Ваш результат: ", result)
			case "3":
				first = input.ReadFloat("Введи первое число: ")
				second = input.ReadFloat("Введи второе число: ")
				result = multiply(first, second)
				fmt.Println("Ваш результат: ", result)
			case "4":
				first = input.ReadFloat("Введите первое число: ")
				second = input.ReadFloat("Введите второе число: ")
				result, err = divide(first, second)
				if err != nil {
        				fmt.Println(err)
        				continue
				} else {
					fmt.Println("Ваш результат: ", result)
				}

			case "5":
				fmt.Println("Выход из программы...")
				break MainLoop


			default:
				fmt.Println("Неизвестная операция")
				continue
		}

		fmt.Println("Продолжить? Напишите *да*  чтобы продолжить")
		var enter string
		fmt.Scan(&enter)

		if strings.ToLower(enter) == "да" || strings.ToLower(enter) == "yes" {
			continue
		} else {
			break
		}

	}
}

package main

import ("fmt"
	"strings"

	input "github.com/goldchell-bra7/gos1mple/input"
)

func inputStr(prompt string) string {
	fmt.Print(prompt)
	var userInput string
	fmt.Scan(&userInput)
	return userInput
}

func convertTemperature(value float64, fromUnit string, toUnit string) float64 {
	if fromUnit == "C" {
		return value * 1.8 + 32
	} else if fromUnit == "F" {
		return (value - 32) / 1.8
	} else if fromUnit == toUnit {
		return value
	} else {
		return 0 // пока что без обработки ошибок
	}
}

func convertWeight(value float64, fromUnit string, toUnit string) float64 {
	if strings.ToLower(fromUnit) == "г" && strings.ToLower(toUnit) == "кг" {
		return value / 1000
	} else if strings.ToLower(fromUnit) == "г" && strings.ToLower(toUnit) == "т" {
		return value / 1000000
	} else if strings.ToLower(fromUnit) == "кг" && strings.ToLower(toUnit) == "г" {
		return value * 1000
	} else if strings.ToLower(fromUnit) == "кг" && strings.ToLower(toUnit) == "т" {
		return value / 1000
	} else if strings.ToLower(fromUnit) == "т" && strings.ToLower(toUnit) == "г" {
		return value * 1000000
	} else if strings.ToLower(fromUnit) == "т" && strings.ToLower(toUnit) == "кг" {
		return value * 1000
	} else {
		return 0 // временно
	}
}

func convertLength(value float64, fromUnit string, toUnit string) float64 {
	switch strings.ToLower(fromUnit) + "->" + strings.ToLower(toUnit) {
		case "см->м":
			return value / 100
		case "см->км":
			return value / 100000
		case "м->км":
			return value / 1000
		case "м->см":
			return value * 100
		case "км->см":
			return value * 100000
		case "км->м":
			return value * 1000
		default:
			if strings.ToLower(fromUnit) == strings.ToLower(toUnit) {
				return value
			}
			return 0
	}
}

func main() {
	var value, result float64
	var fromUnit, toUnit string

	MainLoop:
	for {
		fmt.Println("Привет, ты находишься в меню")
		fmt.Println("Выбери операцию по номеру")
		fmt.Println("1. Температура")
		fmt.Println("2. Вес")
		fmt.Println("3. Длина")
		fmt.Println("4. Выход")
		choice := inputStr("Ваш выбор: ")

		switch choice {
			case "1":
				fromUnit = inputStr("Введите из какой системы хотите перевести: ")
				toUnit = inputStr("Введите в какую систему хотите перевести: ")
				value = input.ReadFloat("Введите число: ")
				fmt.Println("Debug: ", value)
				result = convertTemperature(value, fromUnit, toUnit)
				fmt.Println("Результат: ", result)
			case "2":
				fromUnit = inputStr("Введите из какой величины хотите перевести: ")
				toUnit = inputStr("Введите величину в которую хотите перевести: ")
				value = input.ReadFloat("Введите число: ")
				result = convertWeight(value, fromUnit, toUnit)
				fmt.Println("Результат: ", result)
			case "3":
                                fromUnit = inputStr("Введите из какой величины хотите перевести: ")
                                toUnit = inputStr("Введите величину в которую хотите перевести: ")
                                value = input.ReadFloat("Введите число: ")
                                result = convertLength(value, fromUnit, toUnit)
                                fmt.Println("Результат: ", result)
			case "4":
				break MainLoop
			default:
				fmt.Println("Такой операции нету")
				continue MainLoop
		}
	}
}

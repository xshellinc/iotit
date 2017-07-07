package dialogs

type NumberValidatorFn func(input int) bool

func PositiveNumber(inp int) bool {
	if inp > 0 {
		return true
	}

	return false
}

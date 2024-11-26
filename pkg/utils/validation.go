package utils

import "errors"

func ValidateName(name string) error {
	var length = len(name)
	if length < 2 || length > 20 {
		return errors.New("invalid user input!")
	}

	return nil
}

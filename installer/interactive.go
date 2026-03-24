package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type InteractiveReader struct {
	reader *bufio.Reader
}

func NewInteractiveReader() *InteractiveReader {
	return &InteractiveReader{
		reader: bufio.NewReader(os.Stdin),
	}
}

func (r *InteractiveReader) ReadString(prompt, defaultValue string) (string, error) {
	fmt.Printf("%s [%s]: ", prompt, defaultValue)

	input, err := r.reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue, nil
	}

	return input, nil
}

func (r *InteractiveReader) ReadPassword(prompt string) (string, error) {
	fmt.Printf("%s: ", prompt)

	input, err := r.reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(input), nil
}

func (r *InteractiveReader) Confirm(prompt, defaultValue string) (bool, error) {
	defaultStr := "N"
	if defaultValue == "y" || defaultValue == "Y" {
		defaultStr = "Y"
	}

	fmt.Printf("%s [%s]: ", prompt, defaultStr)

	input, err := r.reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" {
		return defaultValue == "y" || defaultValue == "Y", nil
	}

	return input == "y" || input == "yes", nil
}

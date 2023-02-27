package utils

import (
	"fmt"
	"log"
	"os"
)

func UseTempFile(data string, f func(path string)) {
	file, err := os.CreateTemp("", "*.txt")
	if err != nil {
		panic(fmt.Errorf("couldn't create temp file: %s", err))
	}
	_, err = file.WriteString(data)
	if err != nil {
		panic(fmt.Errorf("couldn't write to temp file: %s", err))
	}
	_ = file.Close()
	filePath := file.Name()
	defer func() {
		err := os.Remove(filePath)
		if err != nil {
			log.Printf("couldn't delete temp file %s: %s", filePath, err)
		}
	}()
	f(filePath)
}

func WriteIntoFile(path string, data string) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = f.Close()
	}()
	if _, err = f.WriteString(data); err != nil {
		panic(err)
	}
	if err = f.Sync(); err != nil {
		panic(err)
	}
}

func ReadFile(path string) string {
	d, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return string(d)
}

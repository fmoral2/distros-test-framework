package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
)

var (
	product *Product
	once    sync.Once
)

type Product struct {
	TFVars  string
	Product string
}

func AddConfigEnv(path string) (*Product, error) {
	once.Do(func() {
		var err error
		product, err = loadEnv(path)
		if err != nil {
			return
		}
	})

	return product, nil
}

func loadEnv(fullPath string) (config *Product, err error) {
	if err = SetEnv(fullPath); err != nil {
		return nil, err
	}

	config = &Product{}
	config.TFVars = os.Getenv("ENV_TFVARS")
	config.Product = os.Getenv("ENV_PRODUCT")
	if config.TFVars == "" || (config.TFVars != "k3s.tfvars" && config.TFVars != "rke2.tfvars") {
		fmt.Printf("unknown tfvars: %s\n", config.TFVars)
		os.Exit(1)
	}

	if config.Product == "" || (config.Product != "k3s" && config.Product != "rke2") {
		fmt.Printf("unknown product: %s\n", config.Product)
		os.Exit(1)
	}

	return config, nil
}

func SetEnv(fullPath string) error {
	file, err := os.Open(fullPath)
	if err != nil {
		fmt.Println("failed to open file:", err)
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		err = os.Setenv(strings.Trim(key, "\""), strings.Trim(value, "\""))
		if err != nil {
			return err
		}
	}

	return scanner.Err()
}

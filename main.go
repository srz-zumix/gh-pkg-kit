/*
Copyright © 2025 srz_zumix
*/
package main

import (
	"github.com/joho/godotenv"
	"github.com/srz-zumix/gh-pkg-kit/cmd"
)

func main() {
	// Load .env file if present; ignore error when not found.
	_ = godotenv.Load()
	cmd.Execute()
}

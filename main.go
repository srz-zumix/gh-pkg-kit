/*
Copyright © 2025 srz_zumix
*/
package main

import (
	"embed"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/srz-zumix/gh-pkg-kit/cmd"
)

//go:embed skills
var skillsFS embed.FS

func main() {
	cmd.RegisterSkillsCmd(skillsFS)
	// Load .env file if present; ignore error when not found.
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		// Log non-NotExist errors to help diagnose configuration issues
		fmt.Fprintln(os.Stderr, "failed to load .env file:", err)
	}
	cmd.Execute()
}

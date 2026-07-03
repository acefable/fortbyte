package main

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a random password",
	RunE: func(cmd *cobra.Command, _ []string) error {
		length, _ := cmd.Flags().GetInt("length")
		noSymbols, _ := cmd.Flags().GetBool("no-symbols")
		pw, err := generatePassword(length, !noSymbols)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), pw)
		return nil
	},
}

func generatePassword(length int, useSymbols bool) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("password length must be positive, got %d", length)
	}
	const alpha = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const symbols = "!@#$%^&*"
	charset := alpha
	if useSymbols {
		charset += symbols
	}
	b := make([]byte, length)
	for i := range b {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		b[i] = charset[idx.Int64()]
	}
	return string(b), nil
}

func init() {
	rootCmd.AddCommand(generateCmd)
	generateCmd.Flags().IntP("length", "l", 24, "Password length")
	generateCmd.Flags().Bool("no-symbols", false, "Exclude special characters")
}

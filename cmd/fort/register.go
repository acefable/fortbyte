package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/youruser/fortbyte/internal/client"
	"github.com/youruser/fortbyte/internal/session"
)

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a new account on the server",
	RunE: func(cmd *cobra.Command, _ []string) error {
		serverURL, err := resolveServerURL(cmd.Flags().Lookup("server").Value.String())
		if err != nil {
			return err
		}

		email, err := promptLine(cmd.OutOrStdout(), os.Stdin, "Email: ")
		if err != nil {
			return fmt.Errorf("read email: %w", err)
		}
		if email == "" {
			return fmt.Errorf("email is required")
		}
		if !strings.Contains(email, "@") {
			return fmt.Errorf("invalid email format")
		}

		fmt.Fprint(cmd.ErrOrStderr(), "Password: ")
		password, err := readPasswordFn()
		if err != nil {
			return fmt.Errorf("read password: %w", err)
		}
		fmt.Fprintln(cmd.ErrOrStderr())

		if len(password) < minPasswordLen {
			return fmt.Errorf("password must be at least %d characters", minPasswordLen)
		}

		fmt.Fprint(cmd.ErrOrStderr(), "Confirm password: ")
		confirm, err := readPasswordFn()
		if err != nil {
			return fmt.Errorf("read confirmation: %w", err)
		}
		fmt.Fprintln(cmd.ErrOrStderr())

		if password != confirm {
			return fmt.Errorf("passwords do not match")
		}

		c := client.New(serverURL)
		tok, err := c.Register(email, password)
		if err != nil {
			return fmt.Errorf("register: %w", err)
		}

		if err := session.StoreTokens(tok.AccessToken, tok.RefreshToken); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not store session tokens: %v\n", err)
		}

		fmt.Fprintln(cmd.OutOrStdout(), styleSuccess.Render("Registered successfully."))
		return nil
	},
}

func init() {
	registerCmd.Flags().String("server", "", "Server URL (overrides config)")
	rootCmd.AddCommand(registerCmd)
}

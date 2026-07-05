package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/youruser/fortbyte/internal/client"
	"github.com/youruser/fortbyte/internal/session"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to the server",
	RunE: func(cmd *cobra.Command, _ []string) error {
		serverURL, err := resolveServerURL(cmd.Flags().Lookup("server").Value.String())
		if err != nil {
			return err
		}

		email, err := promptLine(cmd.OutOrStdout(), cmd.InOrStdin(), "Email: ")
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

		c := client.New(serverURL)
		tok, err := c.Login(email, password)
		if err != nil {
			return fmt.Errorf("login: %w", err)
		}

		if err := session.StoreTokens(tok.AccessToken, tok.RefreshToken); err != nil {
			return fmt.Errorf("store tokens: %w", err)
		}

		fmt.Fprintln(cmd.OutOrStdout(), styleSuccess.Render("Logged in successfully."))
		return nil
	},
}

func init() {
	loginCmd.Flags().String("server", "", "Server URL (overrides config)")
	rootCmd.AddCommand(loginCmd)
}

package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/youruser/fortbyte/internal/client"
	"github.com/youruser/fortbyte/internal/session"
)

var refreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Refresh authentication tokens",
	RunE: func(cmd *cobra.Command, _ []string) error {
		serverURL, err := resolveServerURL(cmd.Flags().Lookup("server").Value.String())
		if err != nil {
			return err
		}

		refreshToken, err := session.LoadRefreshToken()
		if err != nil {
			return fmt.Errorf("no refresh token found: run 'fort login' first")
		}

		c := client.New(serverURL)
		tok, err := c.Refresh(refreshToken)
		if err != nil {
			return fmt.Errorf("refresh: %w", err)
		}

		if err := session.StoreTokens(tok.AccessToken, tok.RefreshToken); err != nil {
			return fmt.Errorf("store tokens: %w", err)
		}

		fmt.Fprintln(cmd.OutOrStdout(), styleSuccess.Render("Tokens refreshed successfully."))
		return nil
	},
}

func init() {
	refreshCmd.Flags().String("server", "", "Server URL (overrides config)")
	rootCmd.AddCommand(refreshCmd)
}

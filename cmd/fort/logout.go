package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/youruser/fortbyte/internal/client"
	"github.com/youruser/fortbyte/internal/session"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out from the server",
	RunE: func(cmd *cobra.Command, _ []string) error {
		serverURL, err := resolveServerURL(cmd.Flags().Lookup("server").Value.String())
		if err != nil {
			// If no server configured, just clear local tokens.
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: %v; clearing local tokens only\n", err)
			if clearErr := session.ClearHostedSession(); clearErr != nil {
				return clearErr
			}
			fmt.Fprintln(cmd.OutOrStdout(), styleSuccess.Render("Logged out (local tokens cleared)."))
			return nil
		}

		accessToken, loadErr := session.LoadAccessToken()
		if loadErr != nil {
			// No token stored — just clear whatever is there.
			if clearErr := session.ClearHostedSession(); clearErr != nil {
				return clearErr
			}
			fmt.Fprintln(cmd.OutOrStdout(), styleSuccess.Render("Logged out."))
			return nil
		}

		c := client.New(serverURL)
		if err := c.Logout(accessToken); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: server logout failed: %v; clearing local tokens\n", err)
		}

		if err := session.ClearHostedSession(); err != nil {
			return err
		}

		fmt.Fprintln(cmd.OutOrStdout(), styleSuccess.Render("Logged out successfully."))
		return nil
	},
}

func init() {
	logoutCmd.Flags().String("server", "", "Server URL (overrides config)")
	rootCmd.AddCommand(logoutCmd)
}

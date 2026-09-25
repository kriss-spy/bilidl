package main

import (
	"fmt"
	"time"

	"bilidown/bilibili"
	"bilidown/cli/credentials"

	"github.com/skip2/go-qrcode"
	"github.com/spf13/cobra"
)

func newAuthCmd(global *globalOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "auth", Short: "Manage Bilibili authentication"}
	var timeout time.Duration
	login := &cobra.Command{Use: "login", Short: "Sign in by scanning a QR code", Long: "Sign in by scanning a QR code with the Bilibili mobile app.\n\nThe resulting session is stored in the local bilidl configuration directory with owner-only permissions.", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		client := &bilibili.BiliClient{}
		info, err := client.NewQRInfo()
		if err != nil {
			return err
		}
		qr, err := qrcode.New(info.URL, qrcode.Medium)
		if err != nil {
			return err
		}
		if !global.quiet {
			if global.json {
				fmt.Fprintln(cmd.ErrOrStderr(), "Open or scan:", info.URL)
			} else {
				printQR(cmd, qr.Bitmap())
				fmt.Fprintln(cmd.OutOrStdout(), "Scan this QR code with the Bilibili mobile app.")
			}
		}
		deadline := time.Now().Add(timeout)
		for time.Now().Before(deadline) {
			status, sessdata, err := client.GetQRStatus(info.QrcodeKey)
			if err != nil {
				return err
			}
			if status.Code == bilibili.QR_SUCCESS {
				if err := credentials.Save(credentials.Credentials{SESSDATA: sessdata}); err != nil {
					return err
				}
				if !global.quiet {
					if global.json {
						return writeJSON(cmd.OutOrStdout(), map[string]bool{"loggedIn": true})
					}
					fmt.Fprintln(cmd.OutOrStdout(), "Logged in.")
				}
				return nil
			}
			if status.Code == bilibili.QR_EXPIRES {
				return fmt.Errorf("QR code expired")
			}
			time.Sleep(2 * time.Second)
		}
		return fmt.Errorf("login timed out")
	}}
	login.Flags().DurationVar(&timeout, "timeout", 3*time.Minute, "stop waiting after this duration")
	status := &cobra.Command{Use: "status", Short: "Show the current login status", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		credential, err := credentials.Load()
		if err != nil {
			return err
		}
		valid := false
		if credential.SESSDATA != "" {
			valid, err = (&bilibili.BiliClient{SESSDATA: credential.SESSDATA}).CheckLogin()
			if err != nil {
				return err
			}
		}
		if global.quiet {
			return nil
		}
		if global.json {
			return writeJSON(cmd.OutOrStdout(), map[string]bool{"loggedIn": valid})
		}
		if !valid {
			fmt.Fprintln(cmd.OutOrStdout(), "Not logged in.")
			return nil
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Logged in.")
		return nil
	}}
	logout := &cobra.Command{Use: "logout", Short: "Remove stored authentication", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if err := credentials.Clear(); err != nil {
			return err
		}
		if !global.quiet {
			if global.json {
				return writeJSON(cmd.OutOrStdout(), map[string]bool{"loggedIn": false})
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Logged out.")
		}
		return nil
	}}
	cmd.AddCommand(login, status, logout)
	return cmd
}

func printQR(cmd *cobra.Command, bitmap [][]bool) {
	for y := 0; y < len(bitmap); y += 2 {
		for x := range bitmap[y] {
			top := bitmap[y][x]
			bottom := y+1 < len(bitmap) && bitmap[y+1][x]
			switch {
			case top && bottom:
				fmt.Fprint(cmd.OutOrStdout(), "█")
			case top:
				fmt.Fprint(cmd.OutOrStdout(), "▀")
			case bottom:
				fmt.Fprint(cmd.OutOrStdout(), "▄")
			default:
				fmt.Fprint(cmd.OutOrStdout(), " ")
			}
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}
}

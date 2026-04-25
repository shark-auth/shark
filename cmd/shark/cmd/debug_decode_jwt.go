package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var debugDecodeJWTCmd = &cobra.Command{
	Use:   "decode-jwt <token>",
	Short: "Decode and pretty-print a JWT (no signature verification)",
	Long: `Decodes a JWT token's header and payload without verifying the signature.
Useful for inspecting claims during development and debugging.

Note: this is a LOCAL operation — no server call is made.

Example:
  shark debug decode-jwt eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9...`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		token := args[0]

		parts := strings.Split(token, ".")
		if len(parts) != 3 {
			return fmt.Errorf("invalid JWT: expected 3 parts separated by '.', got %d", len(parts))
		}

		decodeSegment := func(seg string) (map[string]any, error) {
			// JWT uses base64url without padding — add padding if needed.
			switch len(seg) % 4 {
			case 2:
				seg += "=="
			case 3:
				seg += "="
			}
			data, err := base64.URLEncoding.DecodeString(seg)
			if err != nil {
				return nil, err
			}
			var out map[string]any
			if err := json.Unmarshal(data, &out); err != nil {
				return nil, err
			}
			return out, nil
		}

		header, err := decodeSegment(parts[0])
		if err != nil {
			return fmt.Errorf("decode header: %w", err)
		}
		payload, err := decodeSegment(parts[1])
		if err != nil {
			return fmt.Errorf("decode payload: %w", err)
		}

		if jsonFlag(cmd) {
			return writeJSON(cmd.OutOrStdout(), map[string]any{
				"header":  header,
				"payload": payload,
			})
		}

		fmt.Fprintln(cmd.OutOrStdout(), "header:")
		for k, v := range header {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s: %v\n", k, v)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "payload:")
		for k, v := range payload {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s: %v\n", k, v)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "(signature not verified)")
		return nil
	},
}

func init() {
	addJSONFlag(debugDecodeJWTCmd)
	debugCmd.AddCommand(debugDecodeJWTCmd)
}

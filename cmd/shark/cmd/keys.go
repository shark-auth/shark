package cmd

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

var keysRotate bool

var keysCmd = &cobra.Command{
	Use:   "keys",
	Short: "Manage signing keys",
}

var keysGenerateJWTCmd = &cobra.Command{
	Use:   "generate-jwt",
	Short: "Generate or rotate the RS256 JWT signing keypair",
	Long: `Calls the admin API to rotate the active RS256 JWT signing keypair.

Without --rotate: calls the rotate endpoint (the server always generates and stores
a new key, retiring the current one). Both old and new keys remain in the JWKS
endpoint until the retired key's rotated_at + 2*access_token_ttl has elapsed.

With --rotate: same behaviour — the flag is accepted for compatibility.

Requires a running shark server (--url / SHARK_URL) and admin token (--token / SHARK_ADMIN_TOKEN).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		body, code, err := adminDo(cmd, "POST", "/api/v1/admin/auth/rotate-signing-key", nil)
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}
		if code != http.StatusOK {
			return maybeJSONErr(cmd, "key_rotate_failed", fmt.Errorf("rotate signing key: %s", apiError(body, code)))
		}

		if jsonFlag(cmd) {
			return writeJSON(cmd.OutOrStdout(), body)
		}

		kid, _ := body["kid"].(string)
		algorithm, _ := body["algorithm"].(string)
		status, _ := body["status"].(string)

		fmt.Printf("kid:       %s\n", kid)
		fmt.Printf("algorithm: %s\n", algorithm)
		fmt.Printf("status:    %s\n", status)
		fmt.Println("Rotation complete. Old key(s) marked as retired.")
		return nil
	},
}

func init() {
	keysGenerateJWTCmd.Flags().BoolVar(&keysRotate, "rotate", false, "retire active key(s) and generate a new one (default behaviour; flag kept for compatibility)")
	addJSONFlag(keysGenerateJWTCmd)
	keysCmd.AddCommand(keysGenerateJWTCmd)
}

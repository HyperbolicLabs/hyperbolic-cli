/*
Copyright © 2025 Hyperbolic Labs
*/
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// API client configuration
const (
	baseAPIURL = "https://api.hyperbolic.xyz"
)

// APIClient represents the HTTP client for making API requests
type APIClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewAPIClient creates a new API client
func NewAPIClient() *APIClient {
	return &APIClient{
		baseURL: baseAPIURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Post makes a POST request to the API
func (c *APIClient) Post(path string, body interface{}) (*http.Response, error) {
	fullURL, err := url.JoinPath(c.baseURL, path)
	if err != nil {
		return nil, fmt.Errorf("failed to build URL: %v", err)
	}

	var reqBody string
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %v", err)
		}
		reqBody = string(bodyBytes)
	}

	req, err := http.NewRequest("POST", fullURL, strings.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	return c.httpClient.Do(req)
}

// Get makes a GET request to the API
func (c *APIClient) Get(path string, queryParams map[string]string) (*http.Response, error) {
	fullURL, err := url.JoinPath(c.baseURL, path)
	if err != nil {
		return nil, fmt.Errorf("failed to build URL: %v", err)
	}

	if len(queryParams) > 0 {
		u, err := url.Parse(fullURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse URL: %v", err)
		}

		q := u.Query()
		for key, value := range queryParams {
			q.Add(key, value)
		}
		u.RawQuery = q.Encode()
		fullURL = u.String()
	}

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	return c.httpClient.Do(req)
}

// Response structures for the CLI login API
type TempTokenResponse struct {
	TempToken string `json:"tempToken"`
	LoginURL  string `json:"loginUrl"`
	ExpiresAt string `json:"expiresAt"`
}

type ExchangeTokenResponse struct {
	APIKey string `json:"apiKey"`
	User   struct {
		ID    string  `json:"id"`
		Email *string `json:"email"`
		Name  *string `json:"name"`
	} `json:"user"`
}

type TokenStatusResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// authCmd represents the auth command
var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with Hyperbolic",
	Long:  `Authenticate with your Hyperbolic account using browser-based login or manual API key.`,
}

// loginCmd represents the login subcommand
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login via browser",
	Long:  `Open your browser to login to Hyperbolic and automatically configure the CLI.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := browserLogin(); err != nil {
			fmt.Printf("Error during browser login: %v\n", err)
			return
		}
	},
}

// setKeyCmd represents the set-key subcommand (for manual API key entry)
var setKeyCmd = &cobra.Command{
	Use:   "set-key <api-key>",
	Short: "Set API key manually",
	Long:  `Manually set your Hyperbolic API key. Get your API key from https://app.hyperbolic.ai/settings.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		apiKey := strings.TrimSpace(args[0])

		if apiKey == "" {
			fmt.Println("Error: API key cannot be empty")
			return
		}

		// Create config with the API key
		config := &Config{
			APIKey: apiKey,
		}

		// Save the config
		if err := SaveConfig(config); err != nil {
			fmt.Printf("Error saving configuration: %v\n", err)
			return
		}

		fmt.Println("✓ API key saved successfully!")
		fmt.Println("You can now use other commands like 'hyperbolic rent' without setting environment variables.")
	},
}

// statusCmd represents the status subcommand
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check authentication status",
	Long:  `Check if you are currently authenticated with Hyperbolic.`,
	Run: func(cmd *cobra.Command, args []string) {
		config, err := LoadConfig()
		if err != nil {
			fmt.Println("❌ Not authenticated")
			fmt.Println("Run 'hyperbolic auth login' to authenticate via browser")
			fmt.Println("Or run 'hyperbolic auth set-key <api-key>' to set your API key manually")
			return
		}

		if config.APIKey == "" {
			fmt.Println("❌ No API key found")
			return
		}

		fmt.Println("✓ Authenticated")
		fmt.Printf("API key: %s...%s\n", config.APIKey[:8], config.APIKey[len(config.APIKey)-8:])
	},
}

func browserLogin() error {
	// Initialize API client
	apiClient := NewAPIClient()

	// Step 1: Request temporary token
	tempTokenResp, err := requestTempToken(apiClient)
	if err != nil {
		return fmt.Errorf("failed to request temp token: %v", err)
	}

	// Step 2: Open browser to login URL
	if err := openBrowser(tempTokenResp.LoginURL); err != nil {
		fmt.Printf("Please open this URL in your browser:\n%s\n", tempTokenResp.LoginURL)
	} else {
		fmt.Println("Opening browser for authentication...")
	}

	// Step 3: Poll for authentication completion
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("authentication timed out after 10 minutes")
		default:
			// Check token status
			status, err := checkTokenStatus(apiClient, tempTokenResp.TempToken)
			if err != nil {
				time.Sleep(2 * time.Second)
				continue
			}

			switch status.Status {
			case "authenticated":
				goto authenticated
			case "expired":
				return fmt.Errorf("authentication token expired")
			case "not_found":
				return fmt.Errorf("authentication token not found")
			case "pending":
				// Continue polling
				time.Sleep(2 * time.Second)
				continue
			default:
				time.Sleep(2 * time.Second)
				continue
			}
		}
	}

authenticated:
	// Step 4: Exchange token for API key
	apiKeyResp, err := exchangeToken(apiClient, tempTokenResp.TempToken)
	if err != nil {
		return fmt.Errorf("failed to exchange token: %v", err)
	}

	// Step 5: Save API key
	config := &Config{APIKey: apiKeyResp.APIKey}
	if err := SaveConfig(config); err != nil {
		return fmt.Errorf("failed to save config: %v", err)
	}

	fmt.Printf("Welcome, %s!\n", getDisplayName(apiKeyResp.User))
	fmt.Println("You are now authenticated with Hyperbolic.")
	return nil
}

func requestTempToken(client *APIClient) (*TempTokenResponse, error) {
	resp, err := client.Post("/v2/cli-login/request-token", map[string]interface{}{})
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	var response TempTokenResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &response, nil
}

func checkTokenStatus(client *APIClient, token string) (*TokenStatusResponse, error) {
	queryParams := map[string]string{
		"token": token,
	}

	resp, err := client.Get("/v2/cli-login/status", queryParams)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	var response TokenStatusResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &response, nil
}

func exchangeToken(client *APIClient, tempToken string) (*ExchangeTokenResponse, error) {
	requestBody := map[string]interface{}{
		"tempToken": tempToken,
	}

	resp, err := client.Post("/v2/cli-login/exchange-token", requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	var response ExchangeTokenResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &response, nil
}

func getDisplayName(user struct {
	ID    string  `json:"id"`
	Email *string `json:"email"`
	Name  *string `json:"name"`
}) string {
	if user.Name != nil && *user.Name != "" {
		return *user.Name
	}
	if user.Email != nil && *user.Email != "" {
		return *user.Email
	}
	return user.ID
}

func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(loginCmd)
	authCmd.AddCommand(setKeyCmd)
	authCmd.AddCommand(statusCmd)
}

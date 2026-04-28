package cmd

import (
	"encoding/json"
	"fmt"

	"did_helper/internal/client"

	"github.com/spf13/cobra"
)

var (
	apiURL    string
	apiBody   string
	apiHeader []string
)

// apiCmd is the API command
var apiCmd = &cobra.Command{
	Use:   "api",
	Short: "Send RESTful API requests",
	Long:  "Send HTTP requests to specified API endpoints, supporting GET, POST, PUT, DELETE methods",
}

// apiGetCmd is the GET request command
var apiGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Send GET request",
	RunE: func(cmd *cobra.Command, args []string) error {
		if apiURL == "" {
			return fmt.Errorf("URL is required")
		}

		httpClient := client.NewHTTPClient()
		
		// Add headers
		for _, h := range apiHeader {
			parseHeader(httpClient, h)
		}

		statusCode, body, err := httpClient.Get(apiURL)
		if err != nil {
			return err
		}

		fmt.Printf("Status Code: %d\n", statusCode)
		fmt.Println("Response Body:")
		prettyPrintJSON(body)
		return nil
	},
}

// apiPostCmd is the POST request command
var apiPostCmd = &cobra.Command{
	Use:   "post",
	Short: "Send POST request",
	RunE: func(cmd *cobra.Command, args []string) error {
		if apiURL == "" {
			return fmt.Errorf("URL is required")
		}

		httpClient := client.NewHTTPClient()
		
		// Add headers
		for _, h := range apiHeader {
			parseHeader(httpClient, h)
		}

		var bodyData interface{}
		if apiBody != "" {
			if err := json.Unmarshal([]byte(apiBody), &bodyData); err != nil {
				return fmt.Errorf("Failed to parse request body: %w", err)
			}
		}

		statusCode, respBody, err := httpClient.Post(apiURL, bodyData)
		if err != nil {
			return err
		}

		fmt.Printf("Status Code: %d\n", statusCode)
		fmt.Println("Response Body:")
		prettyPrintJSON(respBody)
		return nil
	},
}

// apiPutCmd is the PUT request command
var apiPutCmd = &cobra.Command{
	Use:   "put",
	Short: "Send PUT request",
	RunE: func(cmd *cobra.Command, args []string) error {
		if apiURL == "" {
			return fmt.Errorf("URL is required")
		}

		httpClient := client.NewHTTPClient()
		
		// Add headers
		for _, h := range apiHeader {
			parseHeader(httpClient, h)
		}

		var bodyData interface{}
		if apiBody != "" {
			if err := json.Unmarshal([]byte(apiBody), &bodyData); err != nil {
				return fmt.Errorf("Failed to parse request body: %w", err)
			}
		}

		statusCode, respBody, err := httpClient.Put(apiURL, bodyData)
		if err != nil {
			return err
		}

		fmt.Printf("Status Code: %d\n", statusCode)
		fmt.Println("Response Body:")
		prettyPrintJSON(respBody)
		return nil
	},
}

// apiDeleteCmd is the DELETE request command
var apiDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Send DELETE request",
	RunE: func(cmd *cobra.Command, args []string) error {
		if apiURL == "" {
			return fmt.Errorf("URL is required")
		}

		httpClient := client.NewHTTPClient()
		
		// Add headers
		for _, h := range apiHeader {
			parseHeader(httpClient, h)
		}

		statusCode, body, err := httpClient.Delete(apiURL)
		if err != nil {
			return err
		}

		fmt.Printf("Status Code: %d\n", statusCode)
		fmt.Println("Response Body:")
		prettyPrintJSON(body)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(apiCmd)

	// Add subcommands
	apiCmd.AddCommand(apiGetCmd)
	apiCmd.AddCommand(apiPostCmd)
	apiCmd.AddCommand(apiPutCmd)
	apiCmd.AddCommand(apiDeleteCmd)

	// Define flags
	apiCmd.PersistentFlags().StringVarP(&apiURL, "url", "u", "", "API URL")
	apiCmd.PersistentFlags().StringVarP(&apiBody, "body", "b", "", "Request body (JSON format)")
	apiCmd.PersistentFlags().StringArrayVarP(&apiHeader, "header", "H", []string{}, "Request header (format: Key:Value)")
}

// parseHeader parses request header
func parseHeader(httpClient *client.HTTPClient, header string) {
	// Simple Key:Value parsing
	for i := 0; i < len(header); i++ {
		if header[i] == ':' && i > 0 && i < len(header)-1 {
			key := header[:i]
			value := header[i+1:]
			httpClient.SetHeader(key, value)
			return
		}
	}
}

// prettyPrintJSON formats and prints JSON
func prettyPrintJSON(jsonStr string) {
	var data interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		// If not JSON, print directly
		fmt.Println(jsonStr)
		return
	}

	prettyJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Println(jsonStr)
		return
	}

	fmt.Println(string(prettyJSON))
}

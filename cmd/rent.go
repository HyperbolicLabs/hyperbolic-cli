/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/spf13/cobra"
)

type RentRequest struct {
	ClusterName string `json:"cluster_name"`
	NodeName    string `json:"node_name"`
	GpuCount    int    `json:"gpu_count"`
	Image       *Image `json:"image,omitempty"`
}

type Image struct {
	Name  string `json:"name"`
	Ports []int  `json:"ports,omitempty"`
}

// rentCmd represents the rent command
var rentCmd = &cobra.Command{
	Use:   "rent [marketplace]",
	Short: "Rent an available GPU instance.",
	Long:  `Rent a GPU instance from a marketplace. Defaults to 'spot' if no marketplace is specified. Run 'hyperbolic rent --help' to view flags. Run 'hyperbolic spot' to view available clusters and nodes.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get marketplace from args, default to "spot"
		marketplace := "spot"
		if len(args) > 0 {
			marketplace = args[0]
		}

		if marketplace == "spot" {
			clusterName, _ := cmd.Flags().GetString("cluster-name")
			nodeName, _ := cmd.Flags().GetString("node-name")
			gpuCount, _ := cmd.Flags().GetInt("gpu-count")
			portStrings, _ := cmd.Flags().GetStringSlice("ports")

			// Validate and process ports
			var ports []int
			if len(portStrings) > 2 {
				fmt.Printf("Error: Maximum of 2 ports can be specified, but %d were provided\n", len(portStrings))
				return
			}

			for _, portStr := range portStrings {
				port, err := strconv.Atoi(portStr)
				if err != nil {
					fmt.Printf("Error: Invalid port number '%s': %v\n", portStr, err)
					return
				}
				if port < 1 || port > 65535 {
					fmt.Printf("Error: Port number '%d' is out of valid range (1-65535)\n", port)
					return
				}
				ports = append(ports, port)
			}

			// Get API key from config file
			apiKey, err := GetAPIKey()
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				fmt.Printf("Please run 'hyperbolic auth YOUR_API_KEY' to save your API key\n")
				fmt.Printf("(Get your API key from https://app.hyperbolic.ai/settings)\n")
				return
			}

			request := RentRequest{
				ClusterName: clusterName,
				NodeName:    nodeName,
				GpuCount:    gpuCount,
			}

			// Only include image if ports are specified
			if len(ports) > 0 {
				request.Image = &Image{
					Name:  "ghcr.io/hyperboliclabs/hyper-dos/sshbox",
					Ports: ports,
				}
			}

			requestBody, err := json.Marshal(request)
			if err != nil {
				fmt.Printf("Error creating request: %v\n", err)
				return
			}

			req, err := http.NewRequest("POST", "https://api.hyperbolic.xyz/v1/marketplace/instances/create", bytes.NewBuffer(requestBody))
			if err != nil {
				fmt.Printf("Error creating HTTP request: %v\n", err)
				return
			}

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+apiKey)

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				fmt.Printf("Error sending request: %v\n", err)
				return
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				fmt.Printf("Error reading response: %v\n", err)
				return
			}

			if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
				fmt.Printf("Error response from API (status code %d): %s\n", resp.StatusCode, string(body))
				return
			}

			fmt.Println("Successfully requested GPU instance:")
			fmt.Println(string(body))
			fmt.Println()
			fmt.Println("To view the status and get the SSH command, run:")
			fmt.Println("  hyperbolic instances")
			if len(ports) > 0 {
				fmt.Println()
				fmt.Println("To view public URLs for exposed ports, run:")
				fmt.Println("  hyperbolic instances <instance-id>")
			}
		} else {
			fmt.Printf("Error: Marketplace '%s' is not supported yet. Only 'spot' is currently supported.\n", marketplace)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(rentCmd)

	// Required flags
	rentCmd.Flags().String("cluster-name", "", "Cluster name for the instance (required)")
	rentCmd.Flags().String("node-name", "", "Node name for the instance (required)")
	rentCmd.Flags().Int("gpu-count", 1, "Number of GPUs to rent (required)")
	
	// Optional flags
	rentCmd.Flags().StringSlice("ports", []string{}, "Ports to expose (up to 2 ports, e.g., --ports 8080,3000 or --ports 8080 --ports 3000)")

	rentCmd.MarkFlagRequired("cluster-name")
	rentCmd.MarkFlagRequired("node-name")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// rentCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// rentCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

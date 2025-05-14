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
	"os"

	"github.com/spf13/cobra"
)

type RentRequest struct {
	ClusterName string `json:"cluster_name"`
	NodeName    string `json:"node_name"`
	GpuCount    int    `json:"gpu_count"`
}

// rentCmd represents the rent command
var rentCmd = &cobra.Command{
	Use:   "rent",
	Short: "Rent an available GPU instance from Hyperbolic",
	Long:  `This allows you to rent a GPU instance from Hyperbolic by specifying the instance ID and the number of GPUs to rent.`,
	Run: func(cmd *cobra.Command, args []string) {
		clusterName, _ := cmd.Flags().GetString("cluster-name")
		nodeName, _ := cmd.Flags().GetString("node-name")
		gpuCount, _ := cmd.Flags().GetInt("gpu-count")

		apiKey := os.Getenv("HYPERBOLIC_API_KEY")
		if apiKey == "" {
			fmt.Println("Error: HYPERBOLIC_API_KEY environment variable is not set")
			return
		}

		request := RentRequest{
			ClusterName: clusterName,
			NodeName:    nodeName,
			GpuCount:    gpuCount,
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
	},
}

func init() {
	rootCmd.AddCommand(rentCmd)

	// Required flags
	rentCmd.Flags().String("cluster-name", "", "Cluster name for the instance (required)")
	rentCmd.Flags().String("node-name", "", "Node name for the instance (required)")
	rentCmd.Flags().Int("gpu-count", 1, "Number of GPUs to rent (required)")

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

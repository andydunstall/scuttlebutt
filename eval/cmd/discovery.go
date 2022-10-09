package cmd

import (
	"context"
	"log"
	"time"

	"github.com/andydunstall/scuttlebutt/cluster"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(discoveryCmd)
}

var discoveryCmd = &cobra.Command{
	Use:   "discovery",
	Short: "Measure the time for nodes in the cluster to discover a new node",
	Run: func(cmd *cobra.Command, args []string) {
		cluster := cluster.NewCluster()
		if err := cluster.AddNodes(32); err != nil {
			log.Fatalf("failed to add nodes: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		if err := cluster.WaitForHealthy(ctx); err != nil {
			log.Fatalf("timed out waiting for cluster to become healthy: %v", err)
		}

		node, err := cluster.AddNode()
		if err != nil {
			log.Fatalf("failed to add node: %v", err)
		}
		if err = cluster.WaitToDiscover(ctx, node.ID); err != nil {
			log.Fatalf("timed out waiting for cluster to discover node: %v", err)
		}
	},
}

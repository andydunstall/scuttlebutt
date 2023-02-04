package cmd

import (
	"context"
	"log"
	"time"

	"github.com/andydunstall/scuttlebutt/eval/pkg/cluster"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(updateCmd)
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Measure the time for an update to propagate to all nodes in the cluster",
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
		node.Gossiper.UpdateLocal("foo", "bar")

		if err = cluster.WaitToUpdate(ctx, node.Gossiper.BindAddr(), "foo", "bar"); err != nil {
			log.Fatalf("timed out waiting for update to propagate: %v", err)
		}
	},
}

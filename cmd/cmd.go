package cmd

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

func Execute() {
	rootCmd := cobra.Command{}
	rootCmd.AddCommand(scrapperCmd)
	rootCmd.AddCommand(fetcherCmd)

	rootCmd.Execute()
}
func OsSignalHandler(cancel context.CancelFunc) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	cancel()
	log.Print("Closing...")
}

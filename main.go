package main

import (
	"log"

	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{Use: "kaschemme"}

	var tcpCmd = &cobra.Command{
		Use:   "tcp",
		Short: "run kaschemme in tcp streaming mode",
		Run:   tcpMode,
	}
	tcpCmd.Flags().String("local-address", "", "local address to bind to")
	tcpCmd.MarkFlagRequired("local-address")
	tcpCmd.Flags().String("remote-address", "", "remote address to foward to")
	tcpCmd.MarkFlagRequired("remote-address")
	rootCmd.AddCommand(tcpCmd)

	var redisCmd = &cobra.Command{
		Use:   "redis",
		Short: "run kaschemme in redis mode",
		Run:   redisMode,
	}
	redisCmd.Flags().String("local-address", "", "local address to bind to")
	redisCmd.MarkFlagRequired("local-address")
	redisCmd.Flags().String("redis-conf", "", "path to redis config")
	redisCmd.MarkFlagRequired("redis-conf")
	rootCmd.AddCommand(redisCmd)

	err := rootCmd.Execute()
	if err != nil {
		log.Fatalln(err)
	}
}

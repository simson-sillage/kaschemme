package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type RedisConfig struct {
	Sentinels []string `yaml:"sentinels"`
}

func redisMode(cmd *cobra.Command, args []string) {
	localAddr, err := cmd.Flags().GetString("local-address")
	if err != nil {
		log.Fatalf("error parsing flags: %s\n", err)
	}
	if _, err := net.ResolveTCPAddr("tcp", localAddr); err != nil {
		log.Fatalf("invalid address: %s: %v\n", localAddr, err)
	}

	configPath, err := cmd.Flags().GetString("redis-conf")
	if err != nil {
		log.Fatalf("error parsing flags: %s\n", err)
	}
	rawConfig, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("error reading config file at %s: %s\n", configPath, err)
	}
	config := RedisConfig{}
	yaml.Unmarshal(rawConfig, &config)
	fmt.Println(config.Sentinels)
}

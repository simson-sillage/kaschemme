package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type RedisConfig struct {
	Sentinels    []string `yaml:"sentinels"`
	RedisPass    string   `yaml:"redis_pass"`
	SentinelPass string   `yaml:"sentinel_pass"`
	MasterName   string   `yaml:"master_name"`
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
	fmt.Println("sentinels:", config.Sentinels)
	fmt.Println("redis pass:", config.RedisPass)
	fmt.Println("sentinel pass:", config.SentinelPass)
	fmt.Println("master name:", config.MasterName)

	ctx := context.TODO()
	masterVoting := make(map[string]int)
	for _, addr := range config.Sentinels {
		sentinel := redis.NewSentinelClient(&redis.Options{
			Addr: addr,
		})
		defer sentinel.Close()
		response, err := sentinel.GetMasterAddrByName(ctx, config.MasterName).Result()
		if err != nil {
			log.Fatalln(err)
		}
		masterAddr := net.JoinHostPort(response[0], response[1])
		masterVoting[masterAddr] += 1
	}
	master := findMaster(masterVoting)
	fmt.Println("master:", master)
}

func findMaster(masterVoting map[string]int) string {
	currentMaster := ""
	currentCount := 0
	for master, count := range masterVoting {
		if count > currentCount {
			currentMaster = master
			currentCount = count
		}
	}

	return currentMaster
}

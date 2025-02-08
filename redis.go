package main

import (
	"context"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type RedisConfig struct {
	Sentinels       []string `yaml:"sentinels"`
	SentinelPass    string   `yaml:"sentinel_pass"`
	SentinelTimeout int      `yaml:"sentinel_timeout"`
	MasterName      string   `yaml:"master_name"`
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
	for _, sentinel := range config.Sentinels {
		if _, err := net.ResolveTCPAddr("tcp", sentinel); err != nil {
			log.Fatalf("invalid address: %s: %v\n", localAddr, err)
		}
	}

	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		log.Fatalf("failed to bind to %s: %s\n", localAddr, err)
	}
	defer listener.Close()
	log.Printf("Listening on: %s\n", localAddr)
	log.Printf("Configured Sentinels: %s\n", strings.Join(config.Sentinels, ", "))

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Connection error: %s\n", err)
			continue
		}
		go redisHandleConnection(conn, config)
	}
}

func redisHandleConnection(src net.Conn, config RedisConfig) {
	candidates := getMasterCandidates(config)
	if len(candidates) <= 0 {
		log.Printf("error: couldn't determine master\n")
		return
	}

	master, votes := findMaster(candidates)
	go handleConnection(src, master)
	log.Printf("current master: %s, votes: %d\n", master, votes)
}

func getMasterCandidates(config RedisConfig) map[string]int {
	candidates := make(map[string]int)
	for _, addr := range config.Sentinels {
		ctx, cancel := context.WithTimeout(
			context.Background(),
			time.Duration(config.SentinelTimeout)*time.Second)
		defer cancel()
		sentinel := redis.NewSentinelClient(&redis.Options{
			Addr: addr,
		})
		defer sentinel.Close()
		// TODO: this should be a go routine. We don't need to query the
		// sentinels sequentially
		response, err := sentinel.GetMasterAddrByName(ctx, config.MasterName).Result()
		if err != nil {
			log.Printf("error getting master from %s: %s\n", addr, err)
			continue
		}
		masterAddr := net.JoinHostPort(response[0], response[1])
		candidates[masterAddr] += 1
	}

	return candidates
}

func findMaster(masterVoting map[string]int) (string, int) {
	currentMaster := ""
	currentCount := 0
	for master, count := range masterVoting {
		if count > currentCount {
			currentMaster = master
			currentCount = count
		}
	}

	return currentMaster, currentCount
}

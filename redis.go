package main

import (
	"context"
	"errors"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type RedisConfig struct {
	Sentinels          []string `yaml:"sentinels"`
	SentinelPass       string   `yaml:"sentinel_pass"`
	SentinelTimeout    int      `yaml:"sentinel_timeout"`
	MasterName         string   `yaml:"master_name"`
	MasterCacheTimeout int      `yaml:"master_cache_timeout"`
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
	// set defaults
	config := RedisConfig{
		SentinelTimeout:    2,
		MasterCacheTimeout: 5,
		SentinelPass:       "",
	}
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

	var ptrMaster atomic.Pointer[string]
	updateMaster(&ptrMaster, config)
	go updateMasterPeriodically(&ptrMaster, config)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Connection error: %s\n", err)
			continue
		}
		go redisHandleConnection(conn, &ptrMaster)
	}
}

func updateMaster(ptrMaster *atomic.Pointer[string], config RedisConfig) {
	master, votes, err := findMaster(config)
	if err != nil {
		log.Printf("error: %s\n", err)
		master = ""
		ptrMaster.Store(&master)
		return
	}
	ptrMaster.Store(&master)
	log.Printf("current master: %s, votes: %d\n", master, votes)
}

func updateMasterPeriodically(ptrMaster *atomic.Pointer[string], config RedisConfig) {
	cacheTimeout := time.Duration(config.MasterCacheTimeout) * time.Second
	log.Printf("trying to update master every %d seconds\n", config.MasterCacheTimeout)
	for {
		updateMaster(ptrMaster, config)
		// should we user time.Ticker here?
		time.Sleep(cacheTimeout)
	}
}

func redisHandleConnection(src net.Conn, ptrMaster *atomic.Pointer[string]) {
	master := *ptrMaster.Load()
	if master == "" {
		log.Printf("error: can't handle connection from %s, master unavailable", src.RemoteAddr().String())
		return
	}

	go handleConnection(src, master)
}

func findMaster(config RedisConfig) (string, int, error) {
	masterVotes := getMasterVotes(config)
	if len(masterVotes) <= 0 {
		return "", 0, errors.New("couldn't determine master")
	}

	master, votes := countMasterVotes(masterVotes)
	return master, votes, nil
}

func getMasterVotes(config RedisConfig) map[string]int {
	var wg sync.WaitGroup
	candidates := make(chan string, len(config.Sentinels))
	for _, addr := range config.Sentinels {
		wg.Add(1)
		go fetchMasterCandidate(addr, config, &wg, candidates)
	}

	votes := make(map[string]int)
	wg.Wait()
	close(candidates)
	for vote := range candidates {
		votes[vote] += 1
	}

	return votes
}

func fetchMasterCandidate(addr string, config RedisConfig, wg *sync.WaitGroup, candidates chan<- string) {
	defer wg.Done()
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(config.SentinelTimeout)*time.Second)
	defer cancel()
	sentinel := redis.NewSentinelClient(&redis.Options{
		Addr: addr,
	})
	defer sentinel.Close()
	response, err := sentinel.GetMasterAddrByName(ctx, config.MasterName).Result()
	if err != nil {
		log.Printf("error getting master from %s: %s\n", addr, err)
		return
	}
	masterAddr := net.JoinHostPort(response[0], response[1])
	candidates <- masterAddr
}

// Master with most votes wins, takes the first on a tie. Maybe err on tie?
// (split brain?)
func countMasterVotes(masterVoting map[string]int) (string, int) {
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

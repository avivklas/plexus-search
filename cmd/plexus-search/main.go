package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/avivklas/plexus"
	"github.com/avivklas/plexus-search/pkg/api"
	"github.com/avivklas/plexus-search/pkg/docstore"
	"github.com/avivklas/plexus-search/pkg/indexstore"
	"github.com/avivklas/plexus-search/pkg/searcher"
)

func main() {
	var (
		nodeID       = flag.String("id", "node-1", "Unique node identifier in the cluster")
		httpAddr     = flag.String("http-addr", "127.0.0.1:8080", "REST HTTP API listen address")
		raftAddr     = flag.String("raft-addr", "127.0.0.1:9000", "Cluster Raft consensus listen address")
		advAddr      = flag.String("advertise-addr", "", "Address advertised to cluster peers for Raft (optional)")
		dataDir      = flag.String("data-dir", "./data", "Directory for Raft consensus log store")
		joinAddrs    = flag.String("join", "", "Comma-separated list of peer RPC addresses to join")
		bootstrap    = flag.Bool("bootstrap", false, "Bootstrap this node as the initial cluster leader")
		syncLog      = flag.Bool("sync-log", false, "Synchronously fsync Raft log appends to disk")
		followerWait = flag.Bool("follower-wait", true, "Wait for follower local FSM apply on writes before returning ACK")
		pprofAddr    = flag.String("pprof-addr", "", "HTTP pprof profile address (e.g. 127.0.0.1:6060)")

		docBackend     = flag.String("doc-store", "mem", "Document storage backend: mem, pebble or s3 (the search index always stays in memory)")
		docPebbleDir   = flag.String("doc-pebble-dir", "", "Pebble directory for -doc-store=pebble (default <data-dir>/docs)")
		docPebbleSync  = flag.Bool("doc-pebble-sync", false, "fsync every Pebble document write")
		docS3Bucket    = flag.String("doc-s3-bucket", "", "S3 bucket for -doc-store=s3")
		docS3Prefix    = flag.String("doc-s3-prefix", "", "S3 key prefix for documents")
		docS3Region    = flag.String("doc-s3-region", "", "S3 region (defaults to AWS config)")
		docS3Endpoint  = flag.String("doc-s3-endpoint", "", "Custom S3 endpoint (MinIO, localstack, ...)")
		docS3PathStyle = flag.Bool("doc-s3-path-style", false, "Use path-style S3 addressing")
	)
	flag.Parse()

	if *pprofAddr != "" {
		go func() {
			log.Printf("Starting pprof HTTP server on %s", *pprofAddr)
			if err := http.ListenAndServe(*pprofAddr, nil); err != nil {
				log.Printf("pprof server error: %v", err)
			}
		}()
	}

	banner := `
   ___  __              ____                  __  
  / _ \/ /__ __ _____  / __/__ ___ ________ _/ /  
 / ___/ / -_) \ / _ \_\ \/ -_) _ '/ __/ __/ _ \ 
/_/  /_/\__/_/_\_//_/___/\__/\_,_/_/  \__/_//_/ 
  Distributed Zero-CDC Search Engine on Plexus Raft
`
	fmt.Println(banner)
	log.Printf("Starting Plexus-Search node...")
	log.Printf("  Node ID:          %s", *nodeID)
	log.Printf("  HTTP API:         %s", *httpAddr)
	log.Printf("  Raft Consensus:   %s", *raftAddr)
	if *advAddr != "" {
		log.Printf("  Advertise Addr:   %s", *advAddr)
	}
	log.Printf("  Data Directory:   %s", *dataDir)
	log.Printf("  Bootstrap Leader: %v", *bootstrap)

	if err := os.MkdirAll(*dataDir, 0755); err != nil {
		log.Fatalf("failed to create data directory: %v", err)
	}

	// 1. Initialize DocumentStore and Bleve IndexStore
	docCfg := docstore.Config{
		Type:   docstore.BackendType(*docBackend),
		Pebble: docstore.PebbleConfig{Dir: *docPebbleDir, Sync: *docPebbleSync},
		S3: docstore.S3Config{
			Bucket:    *docS3Bucket,
			Prefix:    *docS3Prefix,
			Region:    *docS3Region,
			Endpoint:  *docS3Endpoint,
			PathStyle: *docS3PathStyle,
		},
	}
	if docCfg.Type == docstore.BackendPebble && docCfg.Pebble.Dir == "" {
		docCfg.Pebble.Dir = filepath.Join(*dataDir, "docs")
	}
	docStore, err := docstore.Open(docCfg)
	if err != nil {
		log.Fatalf("failed to open document store (%s): %v", *docBackend, err)
	}
	defer docStore.Close()
	indexStore := indexstore.New()
	defer indexStore.Close()

	// 2. Initialize Searcher Coordinator
	s := searcher.New(docStore, indexStore)

	// 3. Configure Plexus Raft Cluster
	clusterCfg := plexus.DefaultClusterConfig(*nodeID, *raftAddr, *dataDir)
	clusterCfg.Bootstrap = *bootstrap
	clusterCfg.ApplyTimeout = 10 * time.Second
	clusterCfg.SyncLog = *syncLog
	clusterCfg.FollowerWaitLocalApply = *followerWait
	if *advAddr != "" {
		clusterCfg.AdvertiseAddr = *advAddr
	}

	if *joinAddrs != "" {
		for _, addr := range strings.Split(*joinAddrs, ",") {
			trimmed := strings.TrimSpace(addr)
			if trimmed != "" {
				clusterCfg.JoinAddrs = append(clusterCfg.JoinAddrs, trimmed)
			}
		}
	}

	cluster, err := plexus.NewCluster(clusterCfg)
	if err != nil {
		log.Fatalf("failed to initialize plexus cluster: %v", err)
	}

	// 4. Register all stores into the cluster FSM
	s.Register(cluster)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 5. Start cluster consensus engine
	if err := cluster.Start(ctx); err != nil {
		log.Fatalf("failed to start cluster: %v", err)
	}
	defer cluster.Stop()

	// 6. Start REST API Server
	apiServer, err := api.NewServer(*httpAddr, s)
	if err != nil {
		log.Fatalf("failed to initialize API server: %v", err)
	}
	if err := apiServer.Start(); err != nil {
		log.Fatalf("failed to start API server: %v", err)
	}
	defer apiServer.Close()

	log.Printf("Plexus-Search is ready!")
	log.Printf("  Index Documents: POST http://%s/api/v1/indexes/{index}/docs", *httpAddr)
	log.Printf("  Search Index:    POST http://%s/api/v1/indexes/{index}/search", *httpAddr)
	log.Printf("  Cluster Status:  GET  http://%s/api/v1/cluster/status", *httpAddr)

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down Plexus-Search...")
}

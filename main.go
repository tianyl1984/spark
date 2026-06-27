package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/tianyl1984/spark/internal/config"
	"github.com/tianyl1984/spark/internal/runner"
	"github.com/tianyl1984/spark/internal/webhook"
)

func main() {
	args := os.Args[1:]
	if len(args) > 0 {
		switch args[0] {
		case "create":
			runCreate(args[1:])
		case "help", "-h", "--help":
			usage()
		default:
			fmt.Fprintf(os.Stderr, "unknown command %q\n\n", args[0])
			usage()
			os.Exit(2)
		}
		return
	}
	runServer()
}

func usage() {
	fmt.Print(`spark — GitHub webhook runner

Usage:
  spark                 start the webhook server
  spark create <name>   scaffold empty config files for a project under $HOME/.spark/<name>
  spark help            show this help
`)
}

// runCreate scaffolds the fixed config files for a project.
func runCreate(args []string) {
	if len(args) != 1 || args[0] == "" {
		fmt.Fprintln(os.Stderr, "usage: spark create <project>")
		os.Exit(2)
	}
	project := args[0]

	sparkDir, err := config.Dir()
	if err != nil {
		log.Fatalf("resolve spark dir: %v", err)
	}

	created, err := runner.Scaffold(sparkDir, project)
	if err != nil {
		log.Fatalf("create project %q: %v", project, err)
	}

	fmt.Printf("project %q ready at %s/%s\n", project, sparkDir, project)
	if len(created) == 0 {
		fmt.Println("all files already existed; nothing to create.")
		return
	}
	for _, p := range created {
		fmt.Printf("  created %s\n", p)
	}
}

// runServer starts the webhook HTTP server.
func runServer() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	sparkDir, err := config.Dir()
	if err != nil {
		log.Fatalf("resolve spark dir: %v", err)
	}

	handler := webhook.New(cfg.Secret, runner.New(sparkDir))

	mux := http.NewServeMux()
	mux.Handle("/", handler)

	addr := fmt.Sprintf(":%d", cfg.Port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("spark listening on %s (config dir %s)", addr, sparkDir)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server: %v", err)
	}
}

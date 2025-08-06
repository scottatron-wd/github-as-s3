package main

import (
	"flag"
	"github-as-s3/internal/application"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func setup() {
	_ = godotenv.Load()
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	// zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

}

func main() {
	setup()

	// Command line flags
	localRepoPath := flag.String("local-repo", "", "Path to local git repository (enables local-only mode)")
	defaultBranch := flag.String("default-branch", "", "Default git branch to use for repositories (defaults to 'master')")
	flag.Parse()

	// Build application options
	var opts []func(*application.Application)

	opts = append(opts, application.WithDefaultGit())

	if *localRepoPath != "" {
		log.Info().Str("local_repo_path", *localRepoPath).Msg("Starting in local repository mode")
		opts = append(opts, application.WithLocalRepoPath(*localRepoPath))
	} else {
		opts = append(opts, application.WithDefaultGithub())
	}

	if *defaultBranch != "" {
		log.Info().Str("default_branch", *defaultBranch).Msg("Using custom default branch")
		opts = append(opts, application.WithDefaultBranch(*defaultBranch))
	}

	app := application.NewApplicationWithOpts(opts...)
	if err := app.Start(); err != http.ErrServerClosed {
		log.Fatal().Err(err)
	}
}

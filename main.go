package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	repository string
	userName   string
	userEmail  string
	targetFile string
	branchName string
	noWeekends bool
	frequency  int
	daysBefore int
	daysAfter  int
	maxCommits int
}

var version = "v0.2.0"

func main() {
	config := parseArgs()

	if config.daysBefore < 0 {
		fmt.Println("days_before must not be negative")
		os.Exit(1)
	}

	if config.daysAfter < 0 {
		fmt.Println("days_after must not be negative")
		os.Exit(1)
	}

	currDate := time.Now()
	directory := "repository-" + currDate.Format("2006-01-02-15-04-05")

	if config.repository != "" {
		start := strings.LastIndex(config.repository, "/") + 1
		end := strings.LastIndex(config.repository, ".")
		if end > start {
			directory = config.repository[start:end]
		}
	}

	// Create directory and initialize git repo
	if err := os.Mkdir(directory, 0755); err != nil {
		fmt.Printf("Error creating directory: %v\n", err)
		os.Exit(1)
	}

	if err := os.Chdir(directory); err != nil {
		fmt.Printf("Error changing directory: %v\n", err)
		os.Exit(1)
	}

	runCommand("git", "init", "-b", config.branchName)

	if config.userName != "" {
		runCommand("git", "config", "user.name", config.userName)
	}

	if config.userEmail != "" {
		runCommand("git", "config", "user.email", config.userEmail)
	}

	// Set start date to 8 PM of the day
	startDate := time.Date(currDate.Year(), currDate.Month(), currDate.Day(), 20, 0, 0, 0, currDate.Location())
	startDate = startDate.AddDate(0, 0, -config.daysBefore)

	// Generate commits for the specified date range
	for i := 0; i < config.daysBefore+config.daysAfter; i++ {
		day := startDate.AddDate(0, 0, i)

		// Check if we should skip weekends
		if config.noWeekends && (day.Weekday() == time.Saturday || day.Weekday() == time.Sunday) {
			continue
		}

		// Random chance based on frequency
		if rand.Intn(100) < config.frequency {
			commitsToday := contributionsPerDay(config.maxCommits)
			for j := 0; j < commitsToday; j++ {
				commitTime := day.Add(time.Duration(j) * time.Minute)
				contribute(commitTime, config.targetFile)
			}
		}
	}

	// Push to remote repository if specified
	if config.repository != "" {
		runCommand("git", "remote", "add", "origin", config.repository)
		runCommand("git", "checkout", "-B", config.branchName) // create if missing, else switch
		runCommand("git", "push", "-u", "origin", config.branchName)
	}

	fmt.Printf("\nRepository generation \x1b[6;30;42mcompleted successfully\x1b[0m!\n")
}

func contribute(date time.Time, targetFile string) {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current directory: %v\n", err)
		return
	}

	readmePath := filepath.Join(cwd, targetFile)

	// Append to README.md
	file, err := os.OpenFile(readmePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return
	}
	defer file.Close()

	message := createMessage(date)
	if _, err := file.WriteString(message + "\n\n"); err != nil {
		fmt.Printf("Error writing to file: %v\n", err)
		return
	}

	// Stage and commit changes
	runCommand("git", "add", ".")
	runCommand("git", "commit", "-m", message, "--date", date.Format("2006-01-02 15:04:05"))
}

func runCommand(name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error running command '%s %s': %v\n", name, strings.Join(args, " "), err)
	}
}

func createMessage(date time.Time) string {
	return date.Format("Contribution: 2006-01-02 15:04")
}

func contributionsPerDay(maxCommits int) int {
	if maxCommits > 20 {
		maxCommits = 20
	}
	if maxCommits < 1 {
		maxCommits = 1
	}
	return rand.Intn(maxCommits) + 1
}

func aliasStringVar(p *string, value string, usage string, names ...string) {
	for _, name := range names {
		flag.StringVar(p, name, value, usage)
	}
}

func aliasIntVar(p *int, value int, usage string, names ...string) {
	for _, name := range names {
		flag.IntVar(p, name, value, usage)
	}
}

func aliasBoolVar(p *bool, value bool, usage string, names ...string) {
	for _, name := range names {
		flag.BoolVar(p, name, value, usage)
	}
}

func parseArgs() Config {
	// Check for -v or --version before any other processing
	for _, arg := range os.Args[1:] {
		if arg == "-v" || arg == "--version" {
			fmt.Printf("contributor version %s\n", version)
			os.Exit(0)
		}
	}

	var config Config

	aliasStringVar(&config.repository, "", "A link to an empty non-initialized remote git repository", "r", "repository")
	aliasStringVar(&config.userName, "", "Overrides user.name git config", "un", "user_name")
	aliasStringVar(&config.userEmail, "", "Overrides user.email git config", "ue", "user_email")
	aliasStringVar(&config.targetFile, "README.md", "The file to write commits into (default: README.md)", "f", "file")
	aliasStringVar(&config.branchName, "contributor", "The branch to create and commit into (default: contributor)", "b", "branch")
	aliasBoolVar(&config.noWeekends, false, "Do not commit on weekends", "nw", "no_weekends")
	aliasIntVar(&config.frequency, 80, "Percentage of days when the script performs commits (default: 80)", "fr", "frequency")
	aliasIntVar(&config.daysBefore, 365, "Number of days before current date to start adding commits (default: 365)", "db", "days_before")
	aliasIntVar(&config.daysAfter, 0, "Number of days after current date until which commits will be added (default: 0)", "da", "days_after")
	aliasIntVar(&config.maxCommits, 10, "Maximum number of commits per day (1-20, default: 10)", "mc", "max_commits")

	// Custom usage message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nGitHub Activity Generator - Creates fake commit history\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s -r https://github.com/user/repo.git -un \"John Doe\" -ue john@example.com\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -nw -fr 60 -db 180 -mc 5\n", os.Args[0])
	}

	flag.Parse()

	// Validate frequency
	if config.frequency < 0 || config.frequency > 100 {
		fmt.Println("frequency must be between 0 and 100")
		os.Exit(1)
	}

	// Validate max commits
	if config.maxCommits < 1 || config.maxCommits > 20 {
		fmt.Println("max_commits must be between 1 and 20")
		os.Exit(1)
	}

	return config
}

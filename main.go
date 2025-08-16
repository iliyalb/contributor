package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
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

type CommitJob struct {
	date       time.Time
	targetFile string
}

var version = "v0.3.1"
var commitMu sync.Mutex

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

	// Sequential Job Processing
	jobs := collectCommitJobs(startDate, config)
	for _, job := range jobs {
		contribute(job.date, job.targetFile)
	}

	// Push to remote repository if specified
	if config.repository != "" {
		runCommand("git", "remote", "add", "origin", config.repository)
		runCommand("git", "checkout", "-B", config.branchName)
		runCommand("git", "push", "-u", "origin", config.branchName)
	}

	fmt.Printf("\nRepository generation \x1b[6;30;42mcompleted successfully\x1b[0m!\n")
}

func collectCommitJobs(startDate time.Time, config Config) []CommitJob {
	var jobs []CommitJob
	totalDays := config.daysBefore + config.daysAfter

	for i := 0; i < totalDays; i++ {
		day := startDate.AddDate(0, 0, i)

		// Skip weekends if requested
		if config.noWeekends && (day.Weekday() == time.Saturday || day.Weekday() == time.Sunday) {
			continue
		}

		// Frequency-based day selection
		if rand.Intn(100) < config.frequency {
			commitsToday := contributionsPerDay(config.maxCommits)
			for j := 0; j < commitsToday; j++ {
				commitTime := day.Add(time.Duration(j) * time.Minute)
				jobs = append(jobs, CommitJob{
					date:       commitTime,
					targetFile: config.targetFile,
				})
			}
		}
	}
	return jobs
}

func contribute(date time.Time, targetFile string) {
	// Atomic file writing and committing
	commitMu.Lock()
	defer commitMu.Unlock()

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current directory: %v\n", err)
		return
	}

	targetPath := filepath.Join(cwd, targetFile)

	// Open and write to the file
	file, err := os.OpenFile(targetPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Error opening file %q: %v\n", targetPath, err)
		return
	}

	message := createMessage(date)
	if _, err := file.WriteString(message + "\n\n"); err != nil {
		_ = file.Close() // best effort
		fmt.Printf("Error writing to file %q: %v\n", targetPath, err)
		return
	}

	// Flush to disk
	if err := file.Close(); err != nil {
		fmt.Printf("Error closing file %q: %v\n", targetPath, err)
		return
	}

	// Stage and commit the change
	if out, err := runCommandWithError("git", "add", "."); err != nil {
		fmt.Printf("git add failed: %v\nOutput:\n%s\n", err, out)
		return
	}
	if out, err := runCommandWithError("git", "commit", "-m", message, "--date", date.Format("2006-01-02 15:04:05")); err != nil {
		// Common helpful hint when commit fails (e.g., duplicate timestamps/messages causing nothing to commit)
		fmt.Printf("git commit failed: %v\nOutput:\n%s\n", err, out)
		return
	}
}

func runCommandWithError(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	// Capture both stdout and stderr
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func runCommand(name string, args ...string) {
	out, err := runCommandWithError(name, args...)
	if err != nil {
		fmt.Printf("Error running command '%s %s': %v\nOutput:\n%s\n", name, strings.Join(args, " "), err, out)
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

package main

import (
	"fmt"
	"bytes"
	"strings"
	"sync"
	"os"
	"os/exec"
	"regexp"
	"encoding/json"

	"github.com/savioxavier/termlink"
	"github.com/cockroachdb/ttycolor"
)

func branchStatus() {
	var branchName string
	var err error

	branchName, err = branch()
	if err != nil {
		fmt.Printf("couldn't get branch name: %s", err)
	}

	ticket, foundTicket := ticket(branchName)

	var wg sync.WaitGroup

	jiraDetailsChan := make(chan string, 1)
	if foundTicket {
		wg.Add(1)
		go func() {
			defer wg.Done()
			jiraDetails(jiraDetailsChan, ticket)
		}()
	} else {
		close(jiraDetailsChan)
	}

	prStatusChan := make(chan pullRequest, 1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		pullRequestStatus(prStatusChan)
	}()


	wg.Wait()

	jiraDetails, isJira := <- jiraDetailsChan
	if isJira  {
		outputTitle("Ticket")
		outputBody(jiraDetails)
	} else {
	}

	outputTitle("Branch")
	if branchName == "" {
		outputBody("(no branch)")
	} else {
		outputBody(branchName)
	}

	if branchName != "" {
		commitDetails, err := getCommitDetails()
		if err != nil {
			fmt.Printf("couldn't get commit details: %s", err)
		}
		if commitDetails != "" {
			outputTitle("Commits")
			outputBody(commitDetails)
		}
	}

	uncommitted, err := getUncommitedFiles()
	if err != nil {
		fmt.Printf("couldn't get commit details: %s", err)
	}
	if uncommitted != "" {
		outputTitle("Uncommitted")
		outputBody(uncommitted)
	}

	// uncommited: git status -s

	pr, isPr := <- prStatusChan
	if isPr {
		outputTitle("Pull Request")
		outputBody(termlink.Link(fmt.Sprintf("pr %d is in %s / %s", pr.Number, pr.State, pr.ReviewDecision), pr.Url))
	}
}

func mainBranch() (string, error) {
    main, err := run([]string{"git", "rev-parse", "--abbrev-ref", "origin/HEAD"})
	if err != nil {
		return "", fmt.Errorf("couldn't get main branch: %v", err)
	}
	return strings.TrimSpace(main), nil
}

func getUncommitedFiles() (string, error) {
    s, err := run([]string{
		"git",
		"status",
		"-s",
	})
	if err != nil {
		return "", fmt.Errorf("couldn't get commit list on this branch: %v", err)
	}
	s = unindentTextRegexp.ReplaceAllString(s, "\n")
	s = strings.TrimLeft(s, " ")
	return s, nil
}

var unindentTextRegexp *regexp.Regexp = regexp.MustCompile(`\n([ \t]+)`)
func getCommitDetails() (string, error) {
    main, err := mainBranch()
	if err != nil {
		return "", err
	}
    commits, err := run([]string{
		"git",
		"log",
		"--format=reference",
		"--color=always",
		fmt.Sprintf("%s..HEAD", main),
	})
	if err != nil {
		return "", fmt.Errorf("couldn't get commit list on this branch: %v", err)
	}
	if commits == "" {
		return "", nil
	}

    filesChanged, err := run([]string{
		"git",
		"diff",
		"--stat",
		main,
		"HEAD",
	})
	if err != nil {
		return "", fmt.Errorf("couldn't get commit changed file list on branch: %v", err)
	}
	filesChanged = unindentTextRegexp.ReplaceAllString(filesChanged, "\n")
	filesChanged = strings.TrimLeft(filesChanged, " ")

	return fmt.Sprintf("%s\n%s", commits, filesChanged), nil
}


func outputTitle(s string) {
	ttycolor.Stdout(ttycolor.Black)
	// bold
	os.Stdout.Write([]byte("\033[0;1;m"))
	fmt.Printf("%s\n", s)
	ttycolor.Stdout(ttycolor.Reset)
}

var indentTextRegexp *regexp.Regexp = regexp.MustCompile(`(\n)`)

func outputBody(s string) {
	indented := indentTextRegexp.ReplaceAllString(s, "\n    ")
	fmt.Printf("    %s\n\n", indented)
}


func run(args []string) (string, error) {
	cmd := exec.Command(args[0], args[1:]...)
	// out, err := cmd.Output() 
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("%v failed with %w; stderr: \n%s", args, err, errb.String())
	}

	return outb.String(), nil
}

// isGitDirectory
//   return code of `git rev-parse --git-dir`

func branch() (string, error) {
	branchOut, err := run([]string{"git", "branch", "--show-current"})
	return strings.TrimSpace(branchOut), err
}

func ticket(branchName string) (string, bool) {
	parts := strings.Split(branchName, "/")
	if len(parts) != 2 {
		return "", false
	} else {
		return parts[0], true
	}
}

func jiraDetails(c chan string, ticket string) {
	details, err := getJiraDetails(ticket)
	if err != nil {
		fmt.Printf("error getting fetching jira ticket %s: %v", ticket, err)
	} else {
		c <- details
	}
	close(c)
}

func getJiraDetails(ticket string) (string, error) {

	details, err := run([]string{"jira", "view", "--template=title", ticket})
	return strings.TrimSpace(details), err
}

type pullRequest struct {
	Id string `json:"id"`
	Number int `json:"number"`
	ReviewDecision string `json:"reviewDecision"`
	State string `json:"state"`
	Url string `json:"url"`
}

func pullRequestStatus(c chan pullRequest) {
	pr, err := prStatus()
	if err != nil {
		fmt.Printf("error fetching the pr: %v", err)
	} else if pr == nil {
		// do nothing--let the channel be closed
	} else {
		c <- *pr
	}
	close(c)
}

func prStatus() (*pullRequest, error) {
	// TODO: distinguish between "no pr" and "error"
	cmd := exec.Command("gh", []string{"pr", "view", "--json", "state,reviewDecision,url,number,id"}...)
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	err := cmd.Run()
	if err != nil {
		if strings.Contains(errb.String(), "no pull requests found") {
			return nil, nil
		} else {
			return nil, fmt.Errorf("error fetching a PR %w; stdout: \n%s stderr: \n%s\n", err, outb.String(), errb.String())
		}
	}

	var state pullRequest
	err = json.Unmarshal(outb.Bytes(), &state)
	if err != nil {
		return nil, fmt.Errorf("error: %w\n", err)
	}

	return &state, nil
}



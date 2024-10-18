package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type pullRequest struct {
	Id             string `json:"id"`
	Number         int    `json:"number"`
	ReviewDecision string `json:"reviewDecision"`
	State          string `json:"state"`
	Url            string `json:"url"`
}

const pullRequestFields = "state,reviewDecision,url,number,id"

func prStatus() (*pullRequest, error) {
	cmd := exec.Command("gh", []string{"pr", "view", "--json", pullRequestFields}...)
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

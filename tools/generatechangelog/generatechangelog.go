// Copyright 2021 FerretDB Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package main contains script that generates changes for the latest version.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"text/template"
	"time"

	"github.com/FerretDB/gh"
	"github.com/google/go-github/v57/github"
	"gopkg.in/yaml.v3"
)

// ReleaseTemplate represents GitHub release template.
type ReleaseTemplate struct {
	Changelog struct {
		Categories []TemplateCategory `yaml:"categories"`
	} `yaml:"changelog"`
}

// TemplateCategory represents a category in the template file.
type TemplateCategory struct {
	Title  string   `yaml:"title"`
	Labels []string `yaml:"labels"`
}

// PRItem represents GitHub PR.
type PRItem struct { //nolint:vet // for readability
	URL    string
	Title  string
	Number int
	User   string
	Labels map[string]struct{}
}

// getMilestone fetches the milestone with the given title.
func getMilestone(ctx context.Context, client *github.Client, milestoneTitle string) (*github.Milestone, error) {
	milestones, _, err := client.Issues.ListMilestones(ctx, "FerretDB", "FerretDB", &github.MilestoneListOptions{
		State: "all",
		ListOptions: github.ListOptions{
			PerPage: 500,
		},
	})
	if err != nil {
		return nil, err
	}

	for _, milestone := range milestones {
		if *milestone.Title == milestoneTitle {
			return milestone, nil
		}
	}

	return nil, fmt.Errorf("no milestone found with the name %s", milestoneTitle)
}

// listMergedPRsOnMilestone returns the list of merged PRs on the given milestone.
func listMergedPRsOnMilestone(ctx context.Context, client *github.Client, milestone *github.Milestone) ([]PRItem, error) {
	issues, _, err := client.Issues.ListByRepo(
		ctx,
		"FerretDB",
		"FerretDB",
		&github.IssueListByRepoOptions{
			State:     "closed",
			Milestone: strconv.Itoa(*milestone.Number),
			Sort:      "created",
			Direction: "asc",
			ListOptions: github.ListOptions{
				PerPage: 500,
			},
		})
	if err != nil {
		return nil, err
	}

	var prItems []PRItem

	for _, issue := range issues {
		if !issue.IsPullRequest() {
			continue
		}

		labels := make(map[string]struct{}, len(issue.Labels))

		for _, label := range issue.Labels {
			labels[*label.Name] = struct{}{}
		}

		prItem := PRItem{
			URL:    *issue.PullRequestLinks.HTMLURL,
			Number: *issue.Number,
			Title:  *issue.Title,
			User:   *issue.User.Login,
			Labels: labels,
		}

		prItems = append(prItems, prItem)
	}

	return prItems, nil
}

// loadReleaseTemplate loads the given release template.
func loadReleaseTemplate(filePath string) (*ReleaseTemplate, error) {
	bytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var tpl ReleaseTemplate

	err = yaml.Unmarshal(bytes, &tpl)
	if err != nil {
		return nil, err
	}

	return &tpl, nil
}

// groupPRsByCategories iterates through the categories and generates Groups of PRs.
func groupPRsByCategories(prItems []PRItem, categories []TemplateCategory) map[string][]PRItem {
	res := map[string][]PRItem{}

	for _, prItem := range prItems {
		var categoryFound bool

		for _, category := range categories {
			for _, label := range category.Labels {
				if _, ok := prItem.Labels[label]; !ok {
					continue
				}

				prs := res[category.Title]
				prs = append(prs, prItem)
				res[category.Title] = prs

				categoryFound = true

				break
			}

			if categoryFound {
				break
			}
		}

		if !categoryFound {
			log.Fatalf("no category found for %q, check the labels in the PR", prItem.URL)
		}
	}

	return res
}

func run(repoRoot, milestoneTitle, previousMilestoneTitle string) {
	releaseYamlFile := filepath.Join(repoRoot, ".github", "release.yml")

	tpl, err := loadReleaseTemplate(releaseYamlFile)
	if err != nil {
		log.Fatalf("Failed to read from template yaml file: %v", err)
	}

	ctx := context.Background()

	client, err := gh.NewRESTClient(os.Getenv("GITHUB_TOKEN"), nil)
	if err != nil {
		log.Fatalf("Failed to create GitHub client: %v", err)
	}

	milestone, err := getMilestone(ctx, client, milestoneTitle)
	if err != nil {
		log.Fatalf("Failed to fetch milestone: %v", err)
	}

	mergedPRs, err := listMergedPRsOnMilestone(ctx, client, milestone)
	if err != nil {
		log.Fatalf("Failed to fetch PRs: %v", err)
	}

	prs := groupPRsByCategories(mergedPRs, tpl.Changelog.Categories)

	templatePath := filepath.Join(repoRoot, "tools", "generatechangelog", "changelog_template.tmpl")

	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		log.Fatalf("Failed to parse template file: %v", err)
	}

	categories := make([]string, len(tpl.Changelog.Categories))
	for i, category := range tpl.Changelog.Categories {
		categories[i] = category.Title
	}

	data := struct { //nolint:vet // for readability
		Date       string
		Current    string
		Previous   string
		URL        string
		Categories []string
		PRs        map[string][]PRItem
	}{
		Date:       time.Now().Format("2006-01-02"),
		Current:    *milestone.Title,       // e.g. v0.8.0 Beta
		Previous:   previousMilestoneTitle, // e.g. v0.7.0
		URL:        *milestone.HTMLURL,
		Categories: categories,
		PRs:        prs,
	}

	if err = tmpl.Execute(os.Stdout, data); err != nil {
		log.Fatalf("Failed to render markdown: %v", err)
	}

	_, _ = fmt.Fprintln(os.Stdout)
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: generatechangelog MILESTONE_TITLE PREVIOUS_MILESTONE_TITLE")
		os.Exit(1)
	}

	repoRoot, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current working directory: %v", err)
	}

	run(repoRoot, os.Args[1], os.Args[2])
}

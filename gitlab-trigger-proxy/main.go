package main

import (
	"net/url"
	"fmt"
	"encoding/json"
	"errors"
	"flag"
	"log"
	"net/http"
	"strings"
)

type server struct {
	config *config
}

func newServer(c *config) *server {
	return &server{config: c}
}

type config struct {
	trackedDirectories stringArrayFlags
	trackedFiles       stringArrayFlags
	triggerToken       string
    webhookSecretToken string
    listenAddress      string
	separator          string
	gitlabHost         string
}

func parseCommandLineArgs() *config {
	c := &config{}
	flag.StringVar(&c.listenAddress, "listen", ":8080", "Listen address")
	flag.StringVar(&c.triggerToken, "trigger-token", "REQUIRED", "REQUIRED - GitLab pipeline trigger token.")
	flag.StringVar(&c.webhookSecretToken, "secret-token", "", "GitLab webhook secret token.")
	flag.StringVar(&c.separator, "separator", ":", "Choose what separates paths in trigger variables value.")
	flag.StringVar(&c.gitlabHost, "gitlab-host", "", "You can set the GitLab host manually. By default the host will be parsed from the projects web URL.")
	flag.Var(&c.trackedFiles, "file", "A file to track for change. Can be set multiple times.")
	flag.Var(&c.trackedDirectories, "directory", "A directory to track for change. Can be set multiple times.")
	flag.Parse()

	return c
}

func main() {
	config := parseCommandLineArgs()
	if config.triggerToken == "REQUIRED" {
		log.Fatal("--trigger-token is required. Exiting now.")
	}
	if config.separator == "" {
		log.Fatal("--separator cannot be empty.")
    }

	http.Handle("/", newServer(config))
	log.Fatal(http.ListenAndServe(config.listenAddress, nil))
}

// ServeHTTP implements the HTTP user interface.
func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// recover runtime errors and log them
	defer func() {
		if err := recover(); err != nil {
			log.Println(err.(error).Error() + " @ " + identifyPanic())
			w.WriteHeader(http.StatusInternalServerError)
		}
    }()
    
	payload, err := parseWebhookRequest(r, s.config.webhookSecretToken)
	if err != nil {
        log.Println(err.Error())
		http.Error(w, err.Error(), 400)
        return
    }
    
    vars := computeTriggerVariables(payload, s.config)
	err = triggerPipeline(s.config.gitlabHost, s.config.triggerToken, payload, vars)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func parseWebhookRequest(r *http.Request, secretToken string) (pushEventPayload, error) {
	// check if secret token matches, if one is set
	if secretToken != "" && (r.Header.Get("x-gitlab-token") != secretToken) {
		return pushEventPayload{}, errors.New("webhook secret token does not match")
    }
    
    var payload pushEventPayload

	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return payload, err
    }
    
    if payload.ObjectKind != "push" {
		return pushEventPayload{}, errors.New("can only handle GitLab push event webhooks")
    }

    if payload.TotalCommitsCount != len(payload.Commits) {
        log.Println("TotalCommitsCount didn't equal the sent commits count. This can happen if more than 20 commits are pushed at once.")
    }

	return payload, nil
}

func computeTriggerVariables(payload pushEventPayload, c *config) map[string][]string {
	changedPaths := getChangedPathsInCommits(payload)
	files, directories := matchChangedToTrackedPaths(changedPaths, c.trackedFiles, c.trackedDirectories)

	return buildTriggerVariables(c.separator, files, directories)
}

// parse all pushed commits and get all paths/files that got changed (added, removed or modified).
func getChangedPathsInCommits(payload pushEventPayload) []string {
	changedFiles := []string{}
	for _, commit := range payload.Commits {
		changedFilesInThisCommit := []string{}
		changedFilesInThisCommit = append(changedFilesInThisCommit, commit.Added...)
		changedFilesInThisCommit = append(changedFilesInThisCommit, commit.Removed...)
		changedFilesInThisCommit = append(changedFilesInThisCommit, commit.Modified...)

		changedFiles = append(changedFiles, changedFilesInThisCommit...)
	}

	return changedFiles
}

func matchChangedToTrackedPaths(changedFiles []string, trackedFiles []string, trackedDirectories []string) ([]string, []string) {
	changedAndTrackedFiles := []string{}
	changedAndTrackedDirs := []string{}

	for _, changedFile := range changedFiles {
		for _, filePath := range trackedFiles {
			if filePath == changedFile {
                changedAndTrackedFiles = append(changedAndTrackedFiles, filePath)
			}
		}
		for _, directoryPath := range trackedDirectories {
			if strings.HasPrefix(changedFile, directoryPath) {
                changedAndTrackedDirs = append(changedAndTrackedDirs, directoryPath)
			}
		}
	}

	return changedAndTrackedFiles, changedAndTrackedDirs
}

func buildTriggerVariables(separator string, files []string, directories []string) map[string][]string {
    result := make(map[string][]string)
    result["variables[FILES_CHANGED]"] = []string{strings.Join(removeDuplicates(files), separator)}
    result["variables[DIRECTORIES_CHANGED]"] = []string{strings.Join(removeDuplicates(directories), separator)}

	return result
}

func triggerPipeline(gitlabHost string, triggerToken string, payload pushEventPayload, formData map[string][]string) error {
    formData["token"] = []string{triggerToken}
    formData["ref"] = []string{payload.Ref}

    webURL, err := url.Parse(payload.Project.Homepage)
    if err != nil {
		log.Println("Couldn't parse GitLab instance URL from webhook payload")
		return err
    }

    if gitlabHost == "" {
        gitlabHost = webURL.Scheme + "://" + webURL.Host
    }
    gitlabTriggerURL := fmt.Sprintf("%v/api/v4/projects/%v/trigger/pipeline", gitlabHost, payload.ProjectID)

    resp, err := http.PostForm(gitlabTriggerURL, formData)
    if err != nil {
		return err
	}
	
	if resp.StatusCode >= http.StatusBadRequest {
		return errors.New("Triggering pipeline failed with " + resp.Status)
	}

	log.Println("Triggered pipeline, response status is " + resp.Status)
	return nil
}

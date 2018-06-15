package main 

import (
	"fmt"
	"strings"
	"runtime"
)

// represents the json payload sent by GitLab push event webhook.
// not needed fields are stripped, for a full list of fields see:
// https://docs.gitlab.com/ce/user/project/integrations/webhooks.html#push-events
type pushEventPayload struct {
	ObjectKind string `json:"object_kind"`
	ProjectID  int    `json:"project_id"`
	Ref        string `json:"ref"`
	Project    struct {
		Name              string `json:"name"`
		Namespace         string `json:"namespace"`
		Homepage          string `json:"homepage"`
		WebURL            string `json:"web_url"`
		PathWithNamespace string `json:"path_with_namespace"`
	} `json:"project"`
	Commits []*struct {
		Added    []string `json:"added"`
		Modified []string `json:"modified"`
		Removed  []string `json:"removed"`
	} `json:"commits"`
	TotalCommitsCount int `json:"total_commits_count"`
}

// stringArrayFlags implements flag.Value .
// Makes it possible to use command line arguments as string slices
type stringArrayFlags []string

func (i *stringArrayFlags) String() string {
	return "a string"
}

func (i *stringArrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func identifyPanic() string {
	var name, file string
	var line int
	var pc [16]uintptr
	
	n := runtime.Callers(3, pc[:])
	for _, pc := range pc[:n] {
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}
		file, line = fn.FileLine(pc)
		name = fn.Name()
		if !strings.HasPrefix(name, "runtime.") {
			break
		}
	}
	
	switch {
	case name != "":
		return fmt.Sprintf("%v:%v", name, line)
	case file != "":
		return fmt.Sprintf("%v:%v", file, line)
	}
	
	return fmt.Sprintf("pc:%x", pc)
}

func removeDuplicates(elements []string) []string {
    encountered := map[string]bool{}
    result := []string{}

    for v := range elements {
        if encountered[elements[v]] == true {
        } else {
            encountered[elements[v]] = true
            result = append(result, elements[v])
        }
    }
    return result
}
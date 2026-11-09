package github

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/flarebyte/gh-arrow/internal/diagnostic"
	"github.com/flarebyte/gh-arrow/internal/listing"
	"github.com/flarebyte/gh-arrow/internal/source"
)

type Discovery struct {
	Repository string `json:"repository"`
	Path       string `json:"path"`
}
type repositoryInfo struct {
	FullName string `json:"full_name"`
	Topics   []string
}

func (c *Client) repositories(ctx context.Context, owner string) ([]repositoryInfo, error) {
	var identity struct{ Type string }
	if err := c.json(ctx, "/users/"+owner, &identity); err != nil {
		return nil, err
	}
	var endpoints []string
	switch identity.Type {
	case "Organization":
		endpoints = []string{"/orgs/" + owner + "/repos?per_page=100&page=%d&type=all"}
	case "User":
		endpoints = []string{"/users/" + owner + "/repos?per_page=100&page=%d&type=owner"}
		if c.token != "" {
			endpoints = append(endpoints, "/user/repos?per_page=100&page=%d&affiliation=owner,collaborator,organization_member&visibility=all")
		}
	default:
		return nil, diagnostic.New("ARROW_NETWORK", "unexpected GitHub owner type")
	}
	seen := map[string]bool{}
	result := []repositoryInfo{}
	for _, endpoint := range endpoints {
		for page := 1; ; page++ {
			if page > 1000 {
				return nil, diagnostic.New("ARROW_INCOMPLETE", "repository enumeration exceeds 1000 pages")
			}
			var repos []repositoryInfo
			if err := c.json(ctx, fmt.Sprintf(endpoint, page), &repos); err != nil {
				return nil, err
			}
			if repos == nil {
				return nil, diagnostic.New("ARROW_NETWORK", "GitHub response is missing its repository array")
			}
			for _, repo := range repos {
				parts := strings.Split(repo.FullName, "/")
				if len(parts) != 2 || !source.ValidRepository(parts[0], parts[1]) {
					return nil, diagnostic.New("ARROW_NETWORK", "invalid repository in GitHub enumeration")
				}
				key := strings.ToLower(repo.FullName)
				if strings.EqualFold(parts[0], owner) && !seen[key] {
					if len(result) >= 100000 {
						return nil, diagnostic.New("ARROW_INCOMPLETE", "discovery exceeds 100000 repositories")
					}
					result = append(result, repo)
					seen[key] = true
				}
			}
			if len(repos) < 100 {
				break
			}
		}
	}
	return result, nil
}
func (c *Client) Discover(ctx context.Context, owner, topic, name string) ([]Discovery, error) {
	if !source.ValidOwner(owner) || name == "" {
		return nil, diagnostic.New("ARROW_USAGE", "discovery requires an owner and exact nonempty path")
	}
	p, err := listing.Parse(name)
	if err != nil {
		return nil, err
	}
	if p.Glob {
		return nil, diagnostic.New("ARROW_USAGE", "discovery does not accept globs")
	}
	repos, err := c.repositories(ctx, owner)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	jobs := make(chan repositoryInfo)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var first error
	matches := []Discovery{}
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for repo := range jobs {
				if ctx.Err() != nil {
					return
				}
				backend, err := c.Repository(ctx, repo.FullName, "")
				if err == nil {
					var entry listing.Entry
					entry, err = backend.Stat(ctx, p.Path)
					if err == nil && entry.Kind == listing.File {
						mu.Lock()
						matches = append(matches, Discovery{repo.FullName, p.Path})
						mu.Unlock()
					}
				}
				if err != nil && !isMissing(err) {
					mu.Lock()
					if first == nil {
						first = fmt.Errorf("check %s:%s: %w", repo.FullName, p.Path, err)
						cancel()
					}
					mu.Unlock()
					return
				}
			}
		}()
	}
send:
	for _, repo := range repos {
		if topic != "" {
			found := false
			for _, t := range repo.Topics {
				if t == topic {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		select {
		case jobs <- repo:
		case <-ctx.Done():
			break send
		}
	}
	close(jobs)
	wg.Wait()
	if first != nil {
		return nil, first
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Repository == matches[j].Repository {
			return matches[i].Path < matches[j].Path
		}
		return matches[i].Repository < matches[j].Repository
	})
	return matches, nil
}

// Sometimes we'd like our Go programs to intelligently
// handle [Unix signals](http://en.wikipedia.org/wiki/Unix_signal).
// For example, we might want a server to gracefully
// shutdown when it receives a `SIGTERM`, or a command-line
// tool to stop processing input if it receives a `SIGINT`.
// Here's how to handle signals in Go with channels.

package main

import (
	"flag"
	"fmt"
	"strings"
	"time"

	"code.google.com/p/goauth2/oauth"
	"github.com/MobileAppTracking/buggerbot/bot"
	"github.com/andybons/hipchat"
	"github.com/google/go-github/github"
)

type PullRequestInfo struct {
	Commits  int
	Comments int
	URL      string
	Title    string
}

func pullRequestBugger(githubAccessToken, user, repo, room string) func() (ch chan bot.BuggerBotProtocol, nextPoll int) {
	pullRequets := map[string]PullRequestInfo{}
	return func() (ch chan bot.BuggerBotProtocol, nextPoll int) {
		beg_day := time.Now().Truncate(24 * time.Hour)
		ch = make(chan bot.BuggerBotProtocol, 100)
		nextPoll = 1200
		t := &oauth.Transport{
			Token: &oauth.Token{AccessToken: githubAccessToken},
		}

		client := github.NewClient(t.Client())

		// list all repositories for the authenticated user
		prs, _, err := client.PullRequests.List(user, repo, &github.PullRequestListOptions{State: "open"})
		if err != nil {
			panic(err)
		}
		for _, pr := range prs {
			comments, _, _ := client.PullRequests.ListComments(user, repo, *pr.Number, nil)
			issue_comments, _, _ := client.Issues.ListComments(user, repo, *pr.Number, nil)
			num_comments := len(comments) + len(issue_comments)

			commits, _, _ := client.PullRequests.ListCommits(user, repo, *pr.Number, nil)
			num_commits := len(commits)
			if val, ok := pullRequets[*pr.URL]; ok {
				if val.Commits < num_commits && num_comments > 0 {
					ch <- bot.BuggerBotProtocol{fmt.Sprintf("New commits added to: %s(%s)", *pr.Title, apiUrlToPubUrl(*pr.URL)), []string{room}}
				}
				val.Commits = num_commits
				val.Comments = num_comments
				pullRequets[val.URL] = val
			} else {
				if pr.CreatedAt.After(beg_day) {
					pullRequets[*pr.URL] = PullRequestInfo{num_commits, num_comments, *pr.URL, *pr.Title}
					ch <- bot.BuggerBotProtocol{
						fmt.Sprintf("Codereview please: %s(%s)",
							*pr.Title,
							apiUrlToPubUrl(*pr.URL)),
						[]string{room}}
				}
			}
		}
		close(ch)
		return
	}
}

func apiUrlToPubUrl(url string) (pub_url string) {
	pub_url = strings.Replace(url, "api.", "", 1)
	pub_url = strings.Replace(pub_url, "/repos", "", 1)
	pub_url = strings.Replace(pub_url, "pulls", "pull", 1)
	return

}

func annoyingBugger() (ch chan bot.BuggerBotProtocol, nextPoll int) {
	ch = make(chan bot.BuggerBotProtocol, 1)
	nextPoll = 10
	ch <- bot.BuggerBotProtocol{"Hello, I'm annoying buggerBot", []string{"Reporting"}}
	close(ch)
	return
}

var (
	githubAccessToken  string
	hipchatAccessToken string // = kingpin.Arg(
)

func main() {
	flag.StringVar(&githubAccessToken, "github_access_token", "<github_access_token>", "access token for github api requests")
	flag.StringVar(&hipchatAccessToken, "hipchat_access_token", "<hipchat_access_token>", "access token for hipchat requests")
	flag.Parse()
	mybot := bot.BuggerBot(hipchat.NewClient(hipchatAccessToken))
	mybot.Register(pullRequestBugger(githubAccessToken, "MobileAppTracking", "reporting_core", "Reporting"))
	mybot.Run()
}

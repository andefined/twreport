package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/andefined/anaconda"
	"github.com/urfave/cli"
)

// Project : Project Struct
type Project struct {
	ConsumerKey       string
	ConsumerSecret    string
	AccessToken       string
	AccessTokenSecret string
	CSV               string
	Output            string
	Column            int
	Block             bool
	NoPrompt          bool
	Debug             bool
}

func main() {
	app := cli.NewApp()
	app.Name = "twreport"
	app.Version = "0.0.2"
	app.Usage = "Batch report / block abusive accounts on Twitter"

	app.Commands = []cli.Command{
		{
			Name:   "report",
			Usage:  "Batch report / block abusive accounts from a csv file",
			Action: Report,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "csv",
					Usage: "Path to input csv `FILE` (eg. ~/Desktop/block-users.csv)",
				},
				cli.StringFlag{
					Name:  "out",
					Usage: "Path to output csv `FILE` (eg. ~/Desktop/blocked-users.csv)",
				},
				cli.IntFlag{
					Name:  "column",
					Usage: "screen_name column",
				},
				cli.BoolFlag{
					Name:  "block",
					Usage: "Block accounts (default: false)",
				},
				cli.BoolFlag{
					Name:  "no-prompt",
					Usage: "Skip confirmation step (default: false)",
				},

				cli.StringFlag{
					Name:  "consumer-key",
					Usage: "Twitter Consumer Key",
				},
				cli.StringFlag{
					Name:  "consumer-secret",
					Usage: "Twitter Consumer Secret",
				},
				cli.StringFlag{
					Name:  "access-token",
					Usage: "Twitter Access Token",
				},
				cli.StringFlag{
					Name:  "access-token-secret",
					Usage: "Twitter Access Secret",
				},

				cli.BoolFlag{
					Name:  "debug",
					Usage: "Will not perform the reporting action (default: false)",
				},
			},
		},
		{
			Name:  "merge",
			Usage: "Remove reported / blocked accounts from original csv file",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "csv",
					Usage: "Path to input csv `FILE` (eg. ~/Desktop/block-users.csv)",
				},
				cli.StringFlag{
					Name:  "out",
					Usage: "Path to output csv `FILE` (eg. ~/Desktop/blocked-users.csv)",
				},
			},
		},
	}
	app.Run(os.Args)
}

// Report : Report Command
func Report(c *cli.Context) error {
	now := time.Now()
	// Assign the flags to Project Struct
	project := &Project{
		CSV:               c.String("csv"),
		Output:            c.String("out"),
		Column:            c.Int("column"),
		Block:             c.Bool("block"),
		NoPrompt:          c.Bool("no-prompt"),
		ConsumerKey:       c.String("consumer-key"),
		ConsumerSecret:    c.String("consumer-secret"),
		AccessToken:       c.String("access-token"),
		AccessTokenSecret: c.String("access-token-secret"),
		Debug:             c.Bool("debug"),
	}
	// Twitter API
	anaconda.SetConsumerKey(project.ConsumerKey)
	anaconda.SetConsumerSecret(project.ConsumerSecret)
	api := anaconda.NewTwitterApi(project.AccessToken, project.AccessTokenSecret)

	// Verify Twitter Credentials
	if _, err := api.VerifyCredentials(); err != nil {
		fmt.Printf("Bad Authorization Tokens. Please refer to https://apps.twitter.com/ for your Access Tokens.\n")
		return nil
	}

	// Open the Input File
	file, err := os.Open(project.CSV)
	if err != nil {
		fmt.Println("Error:", err)
		return nil
	}
	defer file.Close()

	// Read the Input File
	reader := csv.NewReader(file)
	reader.Comma = ','

	// Create an Output File
	// This is usefull for keeping a log of users that you allready reported/blocked
	if project.Output == "" {
		project.Output = "./twreport-" + strconv.FormatInt(now.Unix(), 6) + ".csv"
	}
	output, err := os.OpenFile(project.Output, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		fmt.Println("Error:", err)
		return nil
	}
	defer output.Close()

	// Output File Header
	writer := csv.NewWriter(output)
	writer.Write([]string{"screen_name", "report", "block"})
	writer.Flush()

	for {
		// Read Line By Line
		record, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			fmt.Println("Error:", err)
			return nil
		}
		for j, field := range record {
			if j == project.Column {
				if !project.NoPrompt {
					fmt.Printf("WARNING: Are you sure you want to report user `%s`? (y/n): ", field)
					if PromptConfirm() {
						if !project.Debug {
							user := ReportSpam(api, project, field)
							fmt.Printf("User %s Reported (Blocked: %v).\n", user.ScreenName, project.Block)
							writer.Write([]string{user.ScreenName, "true", strconv.FormatBool(project.Block)})
							writer.Flush()
						}
					}
				} else {
					if !project.Debug {
						user := ReportSpam(api, project, field)
						fmt.Printf("User %s Reported (Blocked: %v).\n", user.ScreenName, project.Block)
						writer.Write([]string{user.ScreenName, "true", strconv.FormatBool(project.Block)})
						writer.Flush()
					}
				}
			}
		}
	}

	return nil
}

// ReportSpam ...
func ReportSpam(api *anaconda.TwitterApi, p *Project, screenName string) anaconda.User {
	v := url.Values{}
	v.Set("perform_block", strconv.FormatBool(p.Block))

	res, err := api.PostUsersReportSpam(screenName, v)
	if err != nil {
		fmt.Printf("You are over the limit for spam reports. Sleeping for 15'.\n")
		Sleep()

		return ReportSpam(api, p, screenName)
	}

	return res
}

// PromptConfirm : Prompt for Confirmation
func PromptConfirm() bool {
	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		log.Fatal(err)
	}
	if strings.ToLower(string(rune(response[0]))) == "y" {
		return true
	}
	if strings.ToLower(string(rune(response[0]))) == "n" {
		return false
	}
	fmt.Printf("Please type y (for yes) or n (for no) and then press enter: ")
	return PromptConfirm()
}

// Sleep : Go to Sleep for 15 Minutes (API Limits)
func Sleep() {
	time.Sleep(15 * time.Minute)
}

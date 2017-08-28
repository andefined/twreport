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

func main() {
	app := cli.NewApp()
	app.Name = "twreport"
	app.Version = "0.0.1"
	app.Usage = "Twitter: Report / Block Users"

	app.Commands = []cli.Command{
		{
			Name:   "report",
			Usage:  "Report / Block Users from File",
			Action: Report,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "csv",
					Usage: "Path to your csv `FILE`",
				},
				cli.IntFlag{
					Name:  "column",
					Usage: "Screen Name Column",
				},
				cli.BoolFlag{
					Name:  "block",
					Usage: "Block Users",
				},
				cli.BoolFlag{
					Name:  "prompt",
					Usage: "Ask Before Reporting",
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
			},
		},
	}
	app.Run(os.Args)
}

// Report : Report Command
func Report(c *cli.Context) error {
	now := time.Now()
	project := &Project{}

	project.CSV = c.String("csv")
	project.Column = c.Int("column")
	project.Block = c.BoolT("block")
	project.Prompt = c.BoolT("prompt")

	project.ConsumerKey = c.String("consumer-key")
	project.ConsumerSecret = c.String("consumer-secret")
	project.AccessToken = c.String("access-token")
	project.AccessTokenSecret = c.String("access-token-secret")

	if project.ConsumerKey == "" || project.ConsumerSecret == "" || project.AccessToken == "" || project.AccessTokenSecret == "" {
		log.Fatal("Consumer key/secret and Access token/secret required")
	}

	anaconda.SetConsumerKey(project.ConsumerKey)
	anaconda.SetConsumerSecret(project.ConsumerSecret)
	api := anaconda.NewTwitterApi(project.AccessToken, project.AccessTokenSecret)
	verify, err := api.VerifyCredentials()

	if err != nil || !verify {
		fmt.Printf("%s. Please refer to %s for your Access Tokens.\n", "Bad Authorization Tokens", "https://apps.twitter.com/")
		return nil
	}

	file, err := os.Open(project.CSV)
	if err != nil {
		fmt.Println("Error:", err)
		return nil
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ','

	output, err := os.OpenFile("./twreport-"+strconv.FormatInt(now.Unix(), 6)+".csv", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		fmt.Println("Error:", err)
		return nil
	}

	defer output.Close()

	writer := csv.NewWriter(output)
	writer.Write([]string{"screen_name", "report", "block"})
	writer.Flush()
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			fmt.Println("Error:", err)
			return nil
		}
		for j, field := range record {
			if j == project.Column {
				v := url.Values{}
				v.Set("perform_block", strconv.FormatBool(project.Block))
				if project.Prompt {
					fmt.Printf("WARNING: Are you sure you want to report user `%s`? (y/n): ", field)
					if PromptConfirm() {
						_, err := api.PostUsersReportSpam(field, v)
						if err != nil {
							fmt.Printf("You are over the limit for spam reports. Sleeping for 15'.\n")
							sleep()
						}
						fmt.Printf("User %s Reported (Blocked: %v).\n", field, project.Block)
						writer.Write([]string{field, "true", strconv.FormatBool(project.Block)})
						writer.Flush()
					}
				} else {
					_, err := api.PostUsersReportSpam(field, v)
					if err != nil {
						fmt.Printf("You are over the limit for spam reports. Sleeping for 15'.\n")
						sleep()
					}
					fmt.Printf("User %s Reported (Blocked: %v).\n", field, project.Block)
					writer.Write([]string{field, "true", strconv.FormatBool(project.Block)})
					writer.Flush()
				}
			}
		}
	}

	return nil
}

// Project : Struct
type Project struct {
	ConsumerKey       string `yaml:"consumer-key"`
	ConsumerSecret    string `yaml:"consumer-secret"`
	AccessToken       string `yaml:"access-token"`
	AccessTokenSecret string `yaml:"access-token-secret"`
	CSV               string `yaml:"csv"`
	Column            int    `yaml:"column"`
	Block             bool   `yaml:"block"`
	Prompt            bool   `yaml:"prompt"`
}

// PromptConfirm : ...
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
	fmt.Printf("Please type yes or no and then press enter: ")
	return PromptConfirm()
}

func sleep() {
	time.Sleep(15 * time.Minute)
}
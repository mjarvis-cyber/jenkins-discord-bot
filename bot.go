package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"golang.org/x/exp/rand"
)

type Bot struct {
	Session *discordgo.Session
	Logger  *log.Logger
}

var (
	JenkinsToken string
	JenkinsURL   string
	Logger       *log.Logger
)

var (
	gifCache     = make(map[string][]string)
	cacheMutex   sync.RWMutex
	cacheTimeout = 60 * time.Minute
	lastFetch    = make(map[string]time.Time)
)

const (
	LogFile = "bot.log"
)

func main() {
	// Open or create the log file
	logFile, err := os.OpenFile(LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println("Error opening log file:", err)
		return
	}
	defer logFile.Close()

	// Set the log output to the file
	Logger = log.New(io.MultiWriter(os.Stdout, logFile), "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)

	// Load environment variables from the .env file
	err = godotenv.Load()
	if err != nil {
		Logger.Println("Error loading .env file:", err)
		return
	}

	// Use the loaded environment variables
	JenkinsToken = os.Getenv("JENKINS_TOKEN")
	JenkinsURL = os.Getenv("JENKINS_URL")
	DiscordToken := os.Getenv("DISCORD_TOKEN")

	discord, err := discordgo.New("Bot " + DiscordToken)
	if err != nil {
		Logger.Println("Error creating Discord session:", err)
		return
	}

	bot := Bot{
		Session: discord,
		Logger:  Logger,
	}

	discord.AddHandler(bot.newMsg)

	err = discord.Open()
	if err != nil {
		Logger.Println("Error opening connection to Discord:", err)
		return
	}

	Logger.Println("Bot is connected to Discord")

	defer discord.Close()

	select {}
}

// This function will be called every time a new message is created on any channel.
func (bot *Bot) newMsg(session *discordgo.Session, message *discordgo.MessageCreate) {
	if message.Author.ID == session.State.User.ID {
		return
	}

	switch {
	case strings.Contains(message.Content, "!steak"):
		session.ChannelMessageSend(message.ChannelID, "time")
		gifURL, err := getGIFURL("steak", 50)
		if err != nil {
			Logger.Println("Failed to fetch reek gif, %s", err)
			return
		}
		session.ChannelMessageSend(message.ChannelID, gifURL)
	case strings.Contains(message.Content, "!reek"):
		session.ChannelMessageSend(message.ChannelID, "Austin TRAN Daniels")
		gifURL, err := getGIFURL("theon-greyjoy-reek", 20)
		if err != nil {
			Logger.Println("Failed to fetch reek gif, %s", err)
			return
		}
		session.ChannelMessageSend(message.ChannelID, gifURL)
	case strings.Contains(message.Content, "!croikey"):
		session.ChannelMessageSend(message.ChannelID, "mayte")
		gifURL, err := getGIFURL("crikey", 20)
		if err != nil {
			Logger.Println("Failed to fetch crikey gif, %s", err)
			return
		}
		session.ChannelMessageSend(message.ChannelID, gifURL)
	case strings.HasPrefix(message.Content, "!gif "):
		term := strings.TrimSpace(strings.TrimPrefix(message.Content, "!gif "))
		if term == "" {
			session.ChannelMessageSend(message.ChannelID, "Usage: !gif <search_term>")
			return
		}
		gifURL, err := getGIFURL(term, 20)
		if err != nil {
			session.ChannelMessageSend(message.ChannelID, fmt.Sprintf("Couldn't fetch GIF for '%s': %v", term, err))
			return
		}
		session.ChannelMessageSend(message.ChannelID, gifURL)
	case strings.Contains(message.Content, "!list"):
		jobList, err := bot.getJenkinsJobList()
		if err != nil {
			session.ChannelMessageSend(message.ChannelID, fmt.Sprintf("Error fetching Jenkins job list: %v", err))
			return
		}
		session.ChannelMessageSend(message.ChannelID, fmt.Sprintf("Jenkins Job List:\n%s", jobList))
	case strings.HasPrefix(message.Content, "!runparams"):
		// Handle !runparams command
		pipelineName, err := bot.runPipelineWithParameters(message.Content)
		if err != nil {
			session.ChannelMessageSend(message.ChannelID, fmt.Sprintf("Error handling !runparams: %v", err))
			return
		}
		session.ChannelMessageSend(message.ChannelID, fmt.Sprintf("Jenkins pipeline '%s' triggered successfully!", pipelineName))
	case strings.HasPrefix(message.Content, "!run"):
		// Extract the pipeline name from the message
		parts := strings.Fields(message.Content)
		if len(parts) < 2 {
			session.ChannelMessageSend(message.ChannelID, "Usage: !run <pipeline_name>")
			return
		}
		pipelineName := strings.Join(parts[1:], " ")

		// Trigger the Jenkins pipeline
		err := bot.triggerJenkinsPipeline(pipelineName)
		if err != nil {
			session.ChannelMessageSend(message.ChannelID, fmt.Sprintf("Error triggering Jenkins pipeline '%s': %v", pipelineName, err))
			return
		}
		session.ChannelMessageSend(message.ChannelID, fmt.Sprintf("Jenkins pipeline '%s' triggered successfully!", pipelineName))
	case strings.HasPrefix(message.Content, "!proceed"):
		// Extract the pipeline name from the message
		parts := strings.Fields(message.Content)
		if len(parts) < 2 {
			session.ChannelMessageSend(message.ChannelID, "Usage: !proceed <pipeline_name>")
			return
		}
		pipelineName := strings.Join(parts[1:], " ")

		// Proceed the Jenkins pipeline
		err := bot.proceedJenkinsPipeline(pipelineName)
		if err != nil {
			session.ChannelMessageSend(message.ChannelID, fmt.Sprintf("Error proceeding Jenkins pipeline '%s': %v", pipelineName, err))
			return
		}
		session.ChannelMessageSend(message.ChannelID, fmt.Sprintf("Jenkins pipeline '%s' proceeded successfully!", pipelineName))
	case strings.HasPrefix(message.Content, "!abort"):
		// Extract the pipeline name from the message
		parts := strings.Fields(message.Content)
		if len(parts) < 2 {
			session.ChannelMessageSend(message.ChannelID, "Usage: !abort <pipeline_name>")
			return
		}
		pipelineName := strings.Join(parts[1:], " ")

		// Abort the Jenkins pipeline
		err := bot.abortJenkinsPipeline(pipelineName)
		if err != nil {
			session.ChannelMessageSend(message.ChannelID, fmt.Sprintf("Error aborting Jenkins pipeline '%s': %v", pipelineName, err))
			return
		}
		session.ChannelMessageSend(message.ChannelID, fmt.Sprintf("Jenkins pipeline '%s' aborted", pipelineName))
	case strings.HasPrefix(message.Content, "!parameters"):
		// Extract the pipeline name from the message
		parts := strings.Fields(message.Content)
		if len(parts) < 2 {
			session.ChannelMessageSend(message.ChannelID, "Usage: !parameters <pipeline_name>")
			return
		}
		pipelineName := strings.Join(parts[1:], " ")

		// Send parameters from last build
		parameters, err := bot.fetchJenkinsJobParameters(pipelineName)
		if err != nil {
			session.ChannelMessageSend(message.ChannelID, fmt.Sprintf("Error fetching parameters for '%s': %v", pipelineName, err))
			return
		}
		session.ChannelMessageSend(message.ChannelID, fmt.Sprintf("Parameters from previous run:%s", parameters))
	case strings.HasPrefix(message.Content, "!help"):
		// Provide help information for each command
		helpMsg := "Available Commands:\n" +
			"!list ---------------------------> Fetches and displays the Jenkins job list\n" +
			"!run <pipeline_name> ---------> Triggers a Jenkins pipeline with the specified name\n" +
			"!proceed <pipeline_name> ----> Proceeds the current stage of a pipeline\n" +
			"!abort <pipeline_name> -------> Aborts the current stage of a pipeline\n" +
			"!parameters <pipeline_name> -> Fetches the parameters from the previous build\n\n" +
			"!runparams\n<pipeline_name\n\nparameterKey parameterValue1\n\nparameterKey2 Parameter value 2"
		session.ChannelMessageSend(message.ChannelID, helpMsg)
	}

}

// getJenkinsJobList retrieves the list of Jenkins jobs, their statuses, and other details.
func (bot *Bot) getJenkinsJobList() (string, error) {
	jobList, err := bot.fetchJenkinsJobs()
	if err != nil {
		return "", err
	}

	var result strings.Builder

	for _, job := range jobList {
		// Replace spaces in job name with %20
		jobName := strings.ReplaceAll(job, " ", "%20")

		// Fetch details for each job
		jobStatus, err := bot.fetchJenkinsJobStatus(jobName)
		Logger.Println("Job Name: ", jobName, "Job Status: ", jobStatus)
		if err != nil {
			Logger.Println("Got some error when getting a job status: ", err)
		}

		// Append formatted job information to the result
		result.WriteString(fmt.Sprintf("%s **%s**\n", jobStatus, job))
	}

	return result.String(), nil
}

// fetchJenkinsJobs retrieves the list of Jenkins job names.
func (bot *Bot) fetchJenkinsJobs() ([]string, error) {
	url := JenkinsURL + "/api/json?tree=jobs[name]"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Set Jenkins authorization header
	authHeader := fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("jenkins:"+JenkinsToken)))
	req.Header.Set("Authorization", authHeader)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request failed with status: %s", resp.Status)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Unmarshal the JSON data
	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	// Extract job names
	jobs, ok := data["jobs"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected format for 'jobs'")
	}

	var jobList []string

	for _, job := range jobs {
		jobMap, ok := job.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("unexpected format for 'job'")
		}

		name, nameOk := jobMap["name"].(string)

		if nameOk {
			jobList = append(jobList, fmt.Sprintf("%s", name))
		}
	}

	return jobList, nil
}

// fetchJenkinsJobStatus retrieves the status and ID of a specific Jenkins job.
func (bot *Bot) fetchJenkinsJobStatus(jobName string) (string, error) {
	url := fmt.Sprintf("%s/job/%s/lastBuild/api/json", JenkinsURL, jobName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	// Set Jenkins authorization header
	authHeader := fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("jenkins:"+JenkinsToken)))
	req.Header.Set("Authorization", authHeader)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "<:jenkinsnotrun:1254459002167885988>", fmt.Errorf("HTTP request failed with status: %s", resp.Status)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Unmarshal the JSON data
	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return "", err
	}

	// Check if the job is in progress
	inProgress, ok := data["inProgress"].(bool)
	if ok && inProgress {
		return "<a:jenkinsrunning:1194478025975279687>", nil
	}

	// If not in progress, return the result
	status, ok := data["result"].(string)
	if !ok {
		return "<:jenkinsnotrun:1254459002167885988>", nil
	}

	// Map Jenkins statuses to Discord emojis
	switch status {
	case "SUCCESS":
		return "<:jenkinsgreencheck:1192251531811094588>", nil
	case "FAILURE":
		return "<:jenkinsfail:1192276960399851641>", nil
	default:
		return "<:jenkinsnotrun:1254459002167885988>", nil
	}
}

// fetchJenkinsJobRunNumber retrieves the ID of a specific Jenkins job.
func (bot *Bot) fetchJenkinsJobRunNumber(jobName string) (int, error) {
	url := fmt.Sprintf("%s/job/%s/lastBuild/api/json", JenkinsURL, jobName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}

	// Set Jenkins authorization header
	authHeader := fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("jenkins:"+JenkinsToken)))
	req.Header.Set("Authorization", authHeader)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("HTTP request failed with status: %s", resp.Status)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	// Log the raw JSON for debugging
	Logger.Println("Raw Jenkins Jobs JSON:", string(body))

	// Unmarshal the JSON data
	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return 0, err
	}

	idStr, idOk := data["id"].(string)
	if !idOk {
		return 0, fmt.Errorf("unable to extract build ID")
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, err
	}

	return int(id), nil
}

// triggerJenkinsPipeline triggers a Jenkins pipeline with optional parameters.
func (bot *Bot) triggerJenkinsPipeline(pipelineName string) error {
	// Attempt to trigger pipeline without parameters
	urlWithoutParams := fmt.Sprintf("%s/job/%s/build", JenkinsURL, pipelineName)
	err := bot.triggerPipelineWithURL(urlWithoutParams)

	if err != nil {
		// If triggering without parameters fails, try triggering with parameters
		urlWithParams := fmt.Sprintf("%s/job/%s/buildWithParameters", JenkinsURL, pipelineName)
		err = bot.triggerPipelineWithURL(urlWithParams)
	}

	return err
}

// triggerPipelineWithURL triggers a Jenkins pipeline with the given URL.
func (bot *Bot) triggerPipelineWithURL(url string) error {
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}

	// Set Jenkins authorization header
	authHeader := fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("jenkins:"+JenkinsToken)))
	req.Header.Set("Authorization", authHeader)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("HTTP request failed with status: %s", resp.Status)
	}

	return nil
}

func (bot *Bot) proceedJenkinsPipeline(pipelineName string) error {
	// Fetch the most recent build status and ID
	jobName := strings.ReplaceAll(pipelineName, " ", "%20")
	jobId, err := bot.fetchJenkinsJobRunNumber(jobName)
	inputIdentifier, err := bot.fetchJenkinsInputIdentifier(jobName, jobId)
	if err != nil {
		return err
	}

	// Construct the URL to proceed the Jenkins pipeline
	url := fmt.Sprintf("%s/job/%s/%d/input/%s/proceedEmpty", JenkinsURL, jobName, jobId, inputIdentifier)
	Logger.Println("Proceed URL: ", url)

	// Perform the HTTP request
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}

	// Set Jenkins authorization header
	authHeader := fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("jenkins:"+JenkinsToken)))
	req.Header.Set("Authorization", authHeader)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check the response status
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("HTTP request failed with status: %s", resp.Status)
	}

	return nil
}

func (bot *Bot) abortJenkinsPipeline(pipelineName string) error {
	// Fetch the most recent build status and ID
	jobName := strings.ReplaceAll(pipelineName, " ", "%20")
	jobId, err := bot.fetchJenkinsJobRunNumber(jobName)
	inputIdentifier, err := bot.fetchJenkinsInputIdentifier(jobName, jobId)
	if err != nil {
		return err
	}

	// Construct the URL to abort the Jenkins pipeline
	url := fmt.Sprintf("%s/job/%s/%d/input/%s/abort", JenkinsURL, jobName, jobId, inputIdentifier)
	Logger.Println("Proceed URL: ", url)

	// Perform the HTTP request
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}

	// Set Jenkins authorization header
	authHeader := fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("jenkins:"+JenkinsToken)))
	req.Header.Set("Authorization", authHeader)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check the response status
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("HTTP request failed with status: %s", resp.Status)
	}

	return nil
}

// fetchJenkinsInputIdentifier retrieves the input identifier for a specific Jenkins job run.
func (bot *Bot) fetchJenkinsInputIdentifier(pipelineName string, runNumber int) (string, error) {
	// Construct the URL to fetch the input identifier
	jobName := strings.ReplaceAll(pipelineName, " ", "%20")
	url := fmt.Sprintf("%s/job/%s/%d/wfapi/pendingInputActions", JenkinsURL, jobName, runNumber)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	// Set Jenkins authorization header
	authHeader := fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("jenkins:"+JenkinsToken)))
	req.Header.Set("Authorization", authHeader)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP request failed to fetch Input Identifier with: %s", resp.Status)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	Logger.Println("Raw Jenkins Jobs JSON:", string(body))

	// Unmarshal the JSON data
	var actions []map[string]interface{}
	err = json.Unmarshal(body, &actions)
	if err != nil {
		return "", err
	}

	// Check if there are any actions
	if len(actions) > 0 {
		// Extract the first action
		action := actions[0]

		// Check if the action has the 'id' field
		if id, idOk := action["id"].(string); idOk {
			return id, nil
		}
	}

	return "", fmt.Errorf("unable to extract input identifier")
}

func (bot *Bot) fetchJenkinsJobParameters(pipelineName string) (string, error) {
	// Fetch the run number for the given pipeline
	jobName := strings.ReplaceAll(pipelineName, " ", "%20")
	runNumber, err := bot.fetchJenkinsJobRunNumber(jobName)
	if err != nil {
		return "", err
	}

	// Construct the URL to fetch Jenkins job parameters
	url := fmt.Sprintf("%s/job/%s/api/json?tree=builds[actions[parameters[name,value]],number]", JenkinsURL, jobName)

	// Perform the HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	// Set Jenkins authorization header
	authHeader := fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("jenkins:"+JenkinsToken)))
	req.Header.Set("Authorization", authHeader)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Check the response status
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP request failed with status: %s", resp.Status)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Unmarshal the JSON data
	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return "", err
	}

	// Extract builds information
	builds, ok := data["builds"].([]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected format for 'builds'")
	}

	// Iterate over builds and find the one with the matching runNumber
	for _, build := range builds {
		buildMap, ok := build.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("unexpected format for 'build'")
		}

		// Extract build number
		buildNumber, numberOk := buildMap["number"].(float64)
		if !numberOk {
			return "", fmt.Errorf("unable to extract build number")
		}

		// Check if the build number matches the runNumber
		if int(buildNumber) == runNumber {
			// Extract actions
			actions, actionsOk := buildMap["actions"].([]interface{})
			if !actionsOk {
				continue // No actions in this build
			}

			// Initialize a buffer to accumulate parameter data
			var buffer strings.Builder

			// Iterate over actions and accumulate parameter data
			for _, action := range actions {
				actionMap, ok := action.(map[string]interface{})
				if !ok {
					continue // Skip if not a valid action
				}

				// Extract parameters
				parameters, parametersOk := actionMap["parameters"].([]interface{})
				if !parametersOk {
					continue // No parameters in this action
				}

				// Accumulate parameter data
				for _, parameter := range parameters {
					parameterMap, ok := parameter.(map[string]interface{})
					if !ok {
						continue // Skip if not a valid parameter
					}

					name, nameOk := parameterMap["name"].(string)
					value, valueOk := parameterMap["value"].(string)

					if nameOk && valueOk {
						// Add parameter data to the buffer
						buffer.WriteString(fmt.Sprintf("\n\n**%s:** \n%s", name, value))
					}
				}
			}

			return buffer.String(), nil
		}
	}

	// If the build with the matching runNumber is not found
	return "", fmt.Errorf("build with runNumber %d not found", runNumber)
}

func (bot *Bot) runPipelineWithParameters(message string) (string, error) {
	// Split the message into lines
	lines := strings.Split(message, "\n")

	// Ensure the message has at least three lines (command, pipeline name, and parameters)
	if len(lines) < 3 {
		return "", fmt.Errorf("invalid message format")
	}

	// Extract pipeline name from the second line
	pipelineName := strings.TrimSpace(lines[1])

	// Extract parameters from the remaining lines
	parameters := make(map[string]string)

	for _, line := range lines[2:] {
		// Trim leading and trailing whitespaces
		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Split the line into key and values
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			return "", fmt.Errorf("invalid parameter format")
		}

		key := parts[0]
		value := parts[1]

		// Append the value to the existing values (if any)
		existingValue, found := parameters[key]
		if found {
			parameters[key] = existingValue + " " + value
		} else {
			parameters[key] = value
		}
	}

	err := bot.triggerJenkinsPipelineParams(pipelineName, parameters)
	if err != nil {
		return "", fmt.Errorf("failed to trigger Jenkins pipeline: %v", err)
	}

	return pipelineName, nil
}

// triggerPipelineWithParameters triggers a Jenkins pipeline with the given parameters.
func (bot *Bot) triggerJenkinsPipelineParams(jobName string, inputJson map[string]string) error {
	// Convert inputJson to an array of objects
	var jsonArray []map[string]string
	for key, value := range inputJson {
		jsonArray = append(jsonArray, map[string]string{key: value})
	}

	// Convert the array to a JSON string
	jsonParams, err := json.Marshal(jsonArray)
	if err != nil {
		return fmt.Errorf("error encoding JSON: %w", err)
	}

	// Convert byte slice to string for better logging
	Logger.Println("json Params: ", string(jsonParams))

	var parameters []map[string]string
	err = json.Unmarshal(jsonParams, &parameters)
	if err != nil {
		return fmt.Errorf("error decoding JSON: %w", err)
	}

	var queryParams []string
	for _, param := range parameters {
		for key, value := range param {
			queryParams = append(queryParams, fmt.Sprintf("%s=%s", key, url.QueryEscape(value)))
		}
	}

	finalURL := fmt.Sprintf("%s/job/%s/buildWithParameters?%s", JenkinsURL, jobName, strings.Join(queryParams, "&"))

	Logger.Println("Final URL: ", finalURL)

	req, err := http.NewRequest("POST", finalURL, nil)
	if err != nil {
		return err
	}

	// Set Jenkins authorization header and content type.
	authHeader := fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte("jenkins:"+JenkinsToken)))
	req.Header.Set("Authorization", authHeader)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Log the response status code
	Logger.Printf("Jenkins API response status code: %s\n", resp.Status)

	// Log the response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %v", err)
	}
	Logger.Printf("Jenkins API response body: %s\n", responseBody)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("HTTP request failed with status: %s", resp.Status)
	}

	return nil
}

func getGIFURL(searchTerm string, limit int) (string, error) {
	apiKey := os.Getenv("GIPHY_KEY")

	rejectIDs := map[string]bool{
		"1JThPpN776F9e": true,
		"A0SDbHUTcClz2": true,
	}

	// Check cache
	cacheMutex.RLock()
	gifs, found := gifCache[searchTerm]
	last, ok := lastFetch[searchTerm]
	cacheMutex.RUnlock()

	if found && ok && time.Since(last) < cacheTimeout && len(gifs) > 0 {
		return gifs[rand.Intn(len(gifs))], nil
	}

	// Fetch from Giphy
	endpoint := fmt.Sprintf("https://api.giphy.com/v1/gifs/search?api_key=%s&q=%s&limit=%d", url.QueryEscape(apiKey), url.QueryEscape(searchTerm), limit)
	resp, err := http.Get(endpoint)
	if err != nil {
		return "", fmt.Errorf("failed to call Giphy API: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Data []struct {
			ID     string `json:"id"`
			Images struct {
				Original struct {
					URL string `json:"url"`
				} `json:"original"`
			} `json:"images"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to parse Giphy response: %w", err)
	}

	var validGIFs []string
	for _, gif := range result.Data {
		if rejectIDs[gif.ID] {
			Logger.Printf("Rejected GIF ID %s (%s)\n", gif.ID, gif.Images.Original.URL)
			continue
		}
		validGIFs = append(validGIFs, gif.Images.Original.URL)
	}

	cacheMutex.Lock()
	gifCache[searchTerm] = validGIFs
	lastFetch[searchTerm] = time.Now()
	cacheMutex.Unlock()

	if len(validGIFs) == 0 {
		return "https://media.giphy.com/media/VbnUQpnihPSIgIXuZv/giphy.gif", nil // fallback
	}

	return validGIFs[rand.Intn(len(validGIFs))], nil
}

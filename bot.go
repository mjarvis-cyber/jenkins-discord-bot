package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

type Bot struct {
	Session *discordgo.Session
	Logger  *log.Logger
}

var (
	JenkinsToken string
	Logger       *log.Logger
)

const (
	JenkinsURL = "http://127.0.0.1:8080"
	LogFile    = "bot.log"
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
	case strings.Contains(message.Content, "!list"):
		jobList, err := bot.getJenkinsJobList()
		if err != nil {
			session.ChannelMessageSend(message.ChannelID, fmt.Sprintf("Error fetching Jenkins job list: %v", err))
			return
		}
		session.ChannelMessageSend(message.ChannelID, fmt.Sprintf("Jenkins Job List:\n%s", jobList))
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
	case strings.HasPrefix(message.Content, "!runparams"):
		// Handle !runparams command
		err := bot.runPipelineWithParameters(message.Content)
		if err != nil {
			session.ChannelMessageSend(message.ChannelID, fmt.Sprintf("Error handling !runparams: %v", err))
			return
		}
		session.ChannelMessageSend(message.ChannelID, "Pipeline triggered with parameters!")
	case strings.HasPrefix(message.Content, "!help"):
		// Provide help information for each command
		helpMsg := "Available Commands:\n" +
			"!list ---------------------------> Fetches and displays the Jenkins job list\n" +
			"!run <pipeline_name> ---------> Triggers a Jenkins pipeline with the specified name\n" +
			"!proceed <pipeline_name> ----> Proceeds the current stage of a pipeline\n" +
			"!abort <pipeline_name> -------> Aborts the current stage of a pipeline\n" +
			"!parameters <pipeline_name> -> Fetches the parameters from the previous build\n\n" +
            "!runparams\n<pipeline_name\n\nparameterKey\nparametervalue1\nparametervalue2\n\nparameterKey2\nparametervalue"
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
            return "", err
        }

        // Append formatted job information to the result
        result.WriteString(fmt.Sprintf("**%s**\nLast Status: %s\n\n", job, jobStatus))
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

    // Check if the job is in progress
    inProgress, ok := data["inProgress"].(bool)
    if ok && inProgress {
        return "RUNNING", nil
    }

    // If not in progress, return the result
    status, ok := data["result"].(string)
    if !ok {
        return "unknown", nil
    }

    return status, nil
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

func (bot *Bot) runPipelineWithParameters(message string) error {
	// Split the message into lines
	lines := strings.Split(message, "\n")

	// Ensure the message has at least three lines (command, pipeline name, and parameters)
	if len(lines) < 3 {
		return fmt.Errorf("invalid message format")
	}

	// Extract pipeline name from the second line
	pipelineName := strings.TrimSpace(lines[1])

	// Extract parameters from the remaining lines
	parameters := make(map[string][]string)
	var currentParamKey string

	for _, line := range lines[2:] {
		// Trim leading and trailing whitespaces
		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Check if the line is a parameter key
		if currentParamKey == "" {
			// Set the current line as the parameter key
			currentParamKey = line
			parameters[currentParamKey] = nil
		} else {
			// Add the line as a parameter value
			parameters[currentParamKey] = append(parameters[currentParamKey], line)
		}
	}

	// Now you have the pipelineName and parameters, you can trigger the Jenkins pipeline
	// Use the Jenkins API to run the pipeline with the specified parameters

	// Example: Trigger pipeline using bot's method (replace with your actual method)
	err := bot.triggerJenkinsPipelineParams(pipelineName, parameters)
	if err != nil {
		return fmt.Errorf("failed to trigger Jenkins pipeline: %v", err)
	}

	return nil
}

// Example method to trigger Jenkins pipeline with parameters
func (bot *Bot) triggerJenkinsPipeline(pipelineName string, parameters map[string][]string) error {
	// TODO: Implement the logic to trigger Jenkins pipeline with parameters
	fmt.Printf("Triggering Jenkins pipeline: %s with parameters: %v\n", pipelineName, parameters)
	return nil
}
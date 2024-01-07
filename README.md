# Jenkins Discord Bot

## Jenkins Setup
* Create a user, named `jenkins`
* Create an API key for said `jenkins` user
* Create a secret string, named `JENKINS_CREDENTIAL_ID` in your jenkins credential store, containing the API key for the jenkins user
* Create a secret string, named `DISCORD_CREDENTIAL_ID` in your jenkins credential store, containing your discord bot key
* Create a secret string, named `JenkinsWebhook` in your jenkins credential store, containing a webhook for a discord channel
* Deploy the pipeline script to Jenkins
* Trigger the pipeline

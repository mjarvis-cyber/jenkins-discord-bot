pipeline {
    agent {
        label 'master'
    }
    environment {
        CUSTOM_WORKSPACE = "$JENKINS_HOME/workspace/discord_bot"
    }
    parameters {
        string(name: 'GIT_REPO', description: 'Specify Git Repo to use', defaultValue: 'git@github.com:OrangeSquirter/jenkins-discord-bot.git')
        string(name: 'BRANCH', description: 'Select the branch you wish to run', defaultValue: 'master')
    }

    stages {
        stage('Run Discord Bot') {
            steps {
                script {
                    dir("${CUSTOM_WORKSPACE}") {
                        // Run the binary using 'script' to create a pseudo-terminal
                        sh "script -q -c './discord_bot' /dev/null &"
                    }
                }
            }
        }
        stage('Wait for Proceed') {
            steps {
                script {
                    dir("${CUSTOM_WORKSPACE}") {
                        // Wait for user input to proceed
                        withCredentials([string(credentialsId: 'JenkinsWebhook', variable: 'Webhook')]) {
                            discordSend title: "Discord Bot", description: "Click 'Proceed' to build new discord bot version", link: env.BUILD_URL, result: currentBuild.currentResult, webhookURL: "${Webhook}"
                        }
                        input message: 'Press "Proceed" to build new bot', submitter: 'user'
                    }
                }
            }
        }
        stage('Build new version') {
            agent {
                docker {
                    image 'golang:latest'
                    args '-v $CUSTOM_WORKSPACE:$CUSTOM_WORKSPACE --entrypoint /bin/sh'
                }
            }
            steps {
                script {
                    sh "tail -f /dev/null &"
                    dir("${CUSTOM_WORKSPACE}") {
                        sh "rm -rf jenkins-discord-bot*"
                        dir("jenkins-discord-bot") {
                            sh "go mod init bot"
                            sh "go get github.com/bwmarrin/discordgo"
                            sh "go get github.com/joho/godotenv"
                            withCredentials([string(credentialsId: 'JENKINS_CREDENTIAL_ID', variable: 'JENKINS_API_TOKEN')]) {
                                // Replace values in the .env file with Jenkins credentials
                                sh "sed -i 's|JENKINS_TOKEN=.*|JENKINS_TOKEN=${JENKINS_API_TOKEN}|' .env"
                            }

                            withCredentials([string(credentialsId: 'DISCORD_CREDENTIAL_ID', variable: 'DISCORD_API_TOKEN')]) {
                                // Replace values in the .env file with Jenkins credentials
                                sh "sed -i 's|DISCORD_TOKEN=.*|DISCORD_TOKEN=${DISCORD_API_TOKEN}|' .env"
                            }

                            // Build the Go program
                            sh "go build -o discord_bot_test"
                        }
                    }
                }
            }
        }
        stage('Test and Stage for Deployment') {
            agent {
                docker {
                    image 'golang:latest'
                    args '-v $CUSTOM_WORKSPACE:$CUSTOM_WORKSPACE --entrypoint /bin/sh'
                }
            }
            steps {
                script {
                    // Run the binary
                    dir("${CUSTOM_WORKSPACE}/jenkins-discord-bot") {
                        sh "touch bot.log"
                        def output = sh(script: "./discord_bot_test &", returnStdout: true).trim()

                        // Wait for the expected output for up to 30 seconds
                        def timeout = 30
                        def startTime = currentBuild.startTimeInMillis
                        def waitForOutput = {
                            while (true) {
                                output = sh(script: "cat bot.log", returnStdout: true).trim()
                                if (output.contains('Bot is connected to Discord')) {
                                    return true
                                } else {
                                    sleep(5)
                                }

                                def elapsedTime = System.currentTimeMillis() - startTime
                                if (elapsedTime > timeout * 1000) {
                                    return false
                                }
                            }
                        }()

                        // Terminate the binary after 30 seconds
                        sh "pkill -f discord_bot_test"

                        // Check if the expected output was received
                        if (!waitForOutput) {
                            error "Expected output 'Bot is connected to Discord' not received within ${timeout} seconds"
                        } else {
                            sh "cp $CUSTOM_WORKSPACE/jenkins-discord-bot/discord_bot_test $CUSTOM_WORKSPACE/discord_bot_tmp"
                            sh "cp $CUSTOM_WORKSPACE/jenkins-discord-bot/.env $CUSTOM_WORKSPACE"
                        }
                    }
                }
            }
        }
    }
    post {
        success {
            script {
                withCredentials([string(credentialsId: 'JenkinsWebhook', variable: 'Webhook')]) {
                    discordSend title: "Discord Bot", description: "Releasing new discord bot version", link: env.BUILD_URL, result: currentBuild.currentResult, webhookURL: "${Webhook}"
                }
                sh "rm -rf $CUSTOM_WORKSPACE/jenkins-discord-bot*"
                if (params.BRANCH in ['master', 'main', 'develop']) {
                    build job: 'Discord Bot', parameters: [ string(name: 'BRANCH', value: 'master'),], wait: false
                }
                sh "cp $CUSTOM_WORKSPACE/discord_bot_tmp $CUSTOM_WORKSPACE/discord_bot"
            }
        }
        failure {
            script {
                withCredentials([string(credentialsId: 'JenkinsWebhook', variable: 'Webhook')]) {
                    discordSend title: "Discord Bot", description: "Rolling bot back to previous version", link: env.BUILD_URL, result: currentBuild.currentResult, webhookURL: "${Webhook}"
                }
                sh "rm -rf $CUSTOM_WORKSPACE/jenkins-discord-bot*"
                if (params.BRANCH in ['master', 'main', 'develop']) {
                    build job: 'Discord Bot', parameters: [ string(name: 'BRANCH', value: 'master'),], wait: false
                }
            }
        }
        always {
            script {
                try {
                    sh 'pkill -f discord_bot'
                } catch (Exception e) {
                    echo "Failed to kill discord_bot process: ${e.getMessage()}"
                }
                sh "cat /dev/null > $CUSTOM_WORKSPACE/bot.log"
            }
        }
    }
}

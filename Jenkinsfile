pipeline {
    agent {
        dockerfile {
            filename 'Dockerfile'
        }
    }
    environment {
        CUSTOM_WORKSPACE = "${JENKINS_HOME}/workspace/${JOB_NAME}"
    }
    parameters {
        string(name: 'GIT_REPO', description: 'Specify Git Repo to use', defaultValue: 'git@github.com:OrangeSquirter/jenkins-discord-bot.git')
        string(name: 'BRANCH', description: 'Select the branch you wish to run', defaultValue: 'master')
    }

    stages {
        stage('Prepare SSH Key') {
            steps {
                script {
                    withCredentials([sshUserPrivateKey(credentialsId: 'ssh-key', keyFileVariable: 'SSH_KEY_FILE')]) {
                        sh "cp ${SSH_KEY_FILE} ${CUSTOM_WORKSPACE}/id_rsa"
                    }
                }
            }
        }
        stage('Build discord bot') {
            steps {
                script {
                    sh "tail -f /dev/null &"
                    dir("${CUSTOM_WORKSPACE}") {
                        sh "mkdir -p ~/.ssh && yes | cp id_rsa ~/.ssh/id_rsa"
                        sh "ssh-keyscan github.com >> ~/.ssh/known_hosts"
                        sh "rm -rf jenkins-discord-bot*"
                        sh "git clone ${params.GIT_REPO} --branch ${params.BRANCH}"
                        dir("jenkins-discord-bot") {
                            sh "pwd"
                            sh "ls -lah"

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
                            sh "go build -o discord_bot"
                        }
                    }
                }
            }
        }
        stage('Deploy bot') {
            steps {
                script {
                    // Run the binary
                    dir("${CUSTOM_WORKSPACE}/jenkins-discord-bot") {
                        sh "touch bot.log"
                        def output = sh(script: "./discord_bot &", returnStdout: true).trim()

                        // Build the bot daily
                        sleep(time:1, unit:'DAYS')
                        

                        // Terminate the binary after 30 seconds
                        sh "pkill -f discord_bot"
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
                sh "rm -rf ${CUSTOM_WORKSPACE}/jenkins-discord-bot*"
                if (params.BRANCH in ['master', 'main', 'develop']) {
                    build job: 'discord-bot', parameters: [string(name: 'BRANCH', value: 'master')], wait: false
                }
            }
        }
        failure {
            script {
                withCredentials([string(credentialsId: 'JenkinsWebhook', variable: 'Webhook')]) {
                    discordSend title: "Discord Bot", description: "Rolling bot back to previous version", link: env.BUILD_URL, result: currentBuild.currentResult, webhookURL: "${Webhook}"
                }
                sh "rm -rf ${CUSTOM_WORKSPACE}/jenkins-discord-bot*"
                if (params.BRANCH in ['master', 'main', 'develop']) {
                    build job: 'discord-bot', parameters: [string(name: 'BRANCH', value: 'master')], wait: false
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
                sh "cat /dev/null > ${CUSTOM_WORKSPACE}/bot.log"
            }
        }
    }
}

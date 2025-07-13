pipeline {
    agent {
        node {
            label 'main'
        }
    }
    environment {
        CUSTOM_WORKSPACE = "${JENKINS_HOME}/workspace/${JOB_NAME}"
    }
    parameters {
        string(name: 'GIT_REPO', description: 'Specify Git Repo to use', defaultValue: 'git@github.com:OrangeSquirter/jenkins-discord-bot.git')
        string(name: 'JENKINS_ENDPOINT', description: 'Specify jenkins endpoint', defaultValue: 'http://jenkins.pizzasec.com:8080')
        string(name: 'BRANCH', description: 'Select the branch you wish to run', defaultValue: 'master')
        string(name: 'PROXMOX_IP', defaultValue: 'cyberops2.pizzasec.com', description: 'ProxMox IP address')
        string(name: 'PROXMOX_NODE', defaultValue: 'cyberops2', description: 'ProxMox to build the box on')
        string(name: 'PROXMOX_POOL', defaultValue: 'Admin', description: 'ProxMox resource pool to assign')
        string(name: 'TEMPLATE', defaultValue: 'ubuntu-24', description: 'Name of the template to use')
        choice(name: 'CORES', choices: ['2'], description: 'Number of cores that will be allocated to the VM')
        choice(name: 'MEMORY', choices: ['2048'], description: 'Memory allocation for the VM in MB')
        string(name: 'STORAGE', defaultValue: '20', description: 'Storage for the VM in GB')
        string(name: 'VM_NAME', defaultValue: 'discord-bot', description: 'Name of the box to build')
        string(name: 'ROLE', defaultValue: 'jenkins', description: 'Why is this box being built')
        choice(name: 'NETWORK', choices: ['vmbr1'], description: 'Network to place the VM on')
        string(name: 'DOCKER_REGISTRY', defaultValue: 'registry.pizzasec.com', description: 'HTTPS endpoint for private docker registry')
    }

    stages {
        stage('Build Parallel') {
            parallel {
                stage('Build Docker Images'){
                    agent {
                        node {
                            label 'main'
                        }
                    }
                    steps {
                        script {
                            def images = build job: 'docker-bake',
                                parameters: [
                                    string(name: 'PROXMOX_IP',      value: params.PROXMOX_IP),
                                    string(name: 'PROXMOX_NODE',    value: params.PROXMOX_NODE),
                                    string(name: 'PROXMOX_POOL',    value: params.PROXMOX_POOL),
                                    string(name: 'TEMPLATE',        value: params.TEMPLATE)
                                ],
                                propagate: true, 
                                wait: true
                        }
                    }
                }
<<<<<<< HEAD
            }
        }
        stage('Build new version') {
            /*
            environment {
                HTTP_PROXY = 'http://zathras:password1!@172.16.0.1:3128'
                HTTPS_PROXY = 'http://zathras:password1!@172.16.0.1:3128'
            }
            */
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
=======
                stage('Build VM'){
                    agent {
                        node {
                            label 'main'
                        }
                    }
                    steps {
                        script {
                            def buildbox = build job: 'box-builder', 
                                parameters: [
                                    string(name: 'PROXMOX_IP',      value: params.PROXMOX_IP),
                                    string(name: 'PROXMOX_NODE',    value: params.PROXMOX_NODE),
                                    string(name: 'PROXMOX_POOL',    value: params.PROXMOX_POOL),
                                    string(name: 'TEMPLATE',        value: params.TEMPLATE),
                                    string(name: 'CORES',           value: params.CORES),
                                    string(name: 'MEMORY',          value: params.MEMORY),
                                    string(name: 'STORAGE',         value: params.STORAGE),
                                    string(name: 'VM_NAME',         value: params.VM_NAME),
                                    string(name: 'ROLE',            value: params.ROLE),
                                    string(name: 'BRANCH',          value: params.BRANCH),
                                    string(name: 'NETWORK',         value: params.NETWORK)
                                ], 
                                propagate: true, 
                                wait: true
>>>>>>> 583b24ecac91231e68664a2b7349ce19e813eab7

                            copyArtifacts(
                                projectName: 'box-builder', 
                                selector: specific("${buildbox.number}"),
                                filter: 'vm_metadata.json'
                            )
                            stash name: 'vm-metadata', includes: "vm_metadata.json"
                        }
                    }
                }
            }
        }
        stage('Deploy') {
            agent {
                node {
                    label 'main'
                }
            }
            steps {
                script {
                    sh "echo DUNZO!!!"
                }
            }
        }
    }
}

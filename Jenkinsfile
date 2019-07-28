@Library('webmakersteve') _

publishedImages = [
  "575393002463.dkr.ecr.us-west-2.amazonaws.com/nottoscale/bot"
]

pipeline {
  agent any

  triggers {
    githubPush()
  }

  stages {
    stage('Build') {
      steps {
        withVersion {
          script {
            dockerHelper.build(this, publishedImages)
          }
        }
      }
    }
    stage('Publish') {
      when {
        branch 'master'  //only run these steps on the master branch
      }

      steps {
        withVersion {
          script {
            dockerHelper.publish(this, publishedImages)
          }
        }
      }
    }
  }
  post {
    always {
      withVersion {
        script {
          dockerHelper.clean(this, publishedImages)
        }
      }
    }
    /*
    success {
      slackSend color: 'good', message: "Build succeeded: $BUILD_URL"
    }
    failure {
      slackSend color: 'good', message: "Build failed: $BUILD_URL"
    }
    aborted {
      slackSend color: 'good', message: "Build aborted: $BUILD_URL"
    }
    */
  }
}

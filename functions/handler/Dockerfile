ARG GOLANG_VERSION=1.21
ARG NODE_VERSION=20.9.0
ARG JENKINS_RUNNER_VERSION=1.0-beta-31

FROM jenkins/jenkinsfile-runner:build-mvncache as jenkinsfilerunner-mvncache

FROM maven:3.8.6-eclipse-temurin-11 as jenkinsfilerunner-build

RUN apt-get update && apt-get install -y unzip git
ENV MAVEN_OPTS=-Dmaven.repo.local=/mavenrepo
COPY --from=jenkinsfilerunner-mvncache /mavenrepo /mavenrepo

RUN git clone https://github.com/jenkinsci/jenkinsfile-runner.git

RUN cd jenkinsfile-runner && git checkout ${JENKINS_RUNNER_VERSION} \
    && mvn clean package --batch-mode -ntp --show-version --errors

# Prepare the Jenkins core
RUN mkdir /app && unzip /jenkinsfile-runner/vanilla-package/target/war/jenkins.war -d /app/jenkins && \
    rm -rf /app/jenkins/scripts /app/jenkins/jsbundles /app/jenkins/css /app/jenkins/images /app/jenkins/help /app/jenkins/WEB-INF/detached-plugins /app/jenkins/WEB-INF/jenkins-cli.jar /app/jenkins/WEB-INF/lib/jna-4.5.2.jar \
    # Delete HPI files and use the archive directories instead
    RUN echo "Optimizing plugins..." && \
    cd /jenkinsfile-runner/vanilla-package/target/plugins && \
    rm -rf *.hpi && \
    for f in * ; do echo "Exploding $f..." && mv "$f" "$f.jpi" ; done;

FROM amazonlinux:2023 as jenkinspluginsmanager
ENV JENKINS_PM_VERSION 2.5.0
ENV JENKINS_PM_URL https://github.com/jenkinsci/plugin-installation-manager-tool/releases/download/${JENKINS_PM_VERSION}/jenkins-plugin-manager-${JENKINS_PM_VERSION}.jar

RUN yum install -y wget
RUN mkdir -p /app/bin \
    && wget $JENKINS_PM_URL -O /app/bin/jenkins-plugin-manager.jar

FROM node:${NODE_VERSION}-alpine3.18 as node

FROM golang:${GOLANG_VERSION}-alpine3.18 as golang
WORKDIR /build
COPY go.mod go.sum ./
COPY cmd/main.go ./
RUN GOOS=linux go build -ldflags="-s -w" -o handler main.go


FROM alpine:3.18.4

# Install git and Java 11 JRE for Jenkins
RUN apk add openjdk11 git 

# Install build tools
## 1. Golang
COPY --from=golang /usr/local/go/ /usr/local/go/
ENV PATH "/usr/local/go/bin:${PATH}"

## 2. NodeJS (and Yarn from the /opt folder)
COPY --from=node /usr/lib /usr/lib
COPY --from=node /usr/local/lib /usr/local/lib
COPY --from=node /usr/local/include /usr/local/include
COPY --from=node /usr/local/bin /usr/local/bin
COPY --from=node /opt /opt 

# Install Jenkins
ENV JAVA_OPTS "--illegal-access=permit $JAVA_OPTS"
ENV JENKINS_UC https://updates.jenkins.io
ENV CASC_JENKINS_CONFIG /usr/share/jenkins/ref/casc

# the line below is for the Jenkins workspace temp folder
RUN mkdir -p /tmp/temp && chmod 777 /tmp/temp 
RUN mkdir -p /app /usr/share/jenkins/ref/plugins /usr/share/jenkins/ref/casc /app/bin \
    && echo "jenkins: {}" >/usr/share/jenkins/ref/casc/jenkins.yaml

COPY --from=jenkinspluginsmanager /app/bin/jenkins-plugin-manager.jar /app/bin/jenkins-plugin-manager.jar
COPY --from=jenkinsfilerunner-build /app/jenkins /app/jenkins
COPY --from=jenkinsfilerunner-build /jenkinsfile-runner/app/target/appassembler /app
COPY --from=jenkinsfilerunner-build /jenkinsfile-runner/vanilla-package/target/plugins /usr/share/jenkins/ref/plugins

ENV PATH "/app/bin:${PATH}"

# Install lambda handler
COPY --from=golang /build/handler /handler

ENTRYPOINT [ "/handler" ]
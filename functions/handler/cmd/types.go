package main

type Branch struct {
	Name string `yaml:"name"`
}

type RemoteConfig struct {
	Url string `yaml:"url"`
}

type GitConfig struct {
	UserRemoteConfigs []RemoteConfig `yaml:"userRemoteConfigs"`
	Branches          []Branch       `yaml:"branches"`
	GitTool           string         `yaml:"gitTool"`
}

type ScmConfig struct {
	Git GitConfig `yaml:"git"`
}

type RootConfig struct {
	Scm ScmConfig `yaml:"scm"`
}

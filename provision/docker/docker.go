// Copyright 2013 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package docker

import (
	"encoding/json"
	"errors"
	"github.com/globocom/config"
	"github.com/globocom/tsuru/fs"
	"github.com/globocom/tsuru/log"
	"os/exec"
    "github.com/dotcloud/docker"
)

var fsystem fs.Fs

func filesystem() fs.Fs {
	if fsystem == nil {
		fsystem = fs.OsFs{}
	}
	return fsystem
}

// container represents an docker container with the given name.
type container struct {
	name       string
	instanceId string
}

// runCmd executes commands and log the given stdout and stderror.
func runCmd(cmd string, args ...string) (err error, output string) {
	command := exec.Command(cmd, args...)
	out, err := command.CombinedOutput()
	log.Printf("running the cmd: %s with the args: %s", cmd, args)
	output = string(out)
	return err, output
}

// ip returns the ip for the container.
func (c *container) ip() (err error, ip string) {
	docker, err := config.GetString("docker:binary")
	if err != nil {
		return err, ""
	}
	log.Printf("Getting ipaddress to instance %s", c.instanceId)
	err, instance_json := runCmd("sudo", docker, "inspect", c.instanceId)
	if err != nil {
		log.Printf("error(%s) trying to inspect docker instance(%s) to get ipaddress", err, c.instanceId)
		return err, ""
	} else if instance_json == "" {
        log.Printf("error: empty json returned for instance(%s) while trying to get ipaddress", c.instanceId)
        //TODO: provide better error code
		return err, ""
    }
	var jsonBlob = []byte(instance_json)
	var result map[string]interface{}
	err2 := json.Unmarshal(jsonBlob, &result)
	if err2 != nil {
        log.Printf("error(%s) parsing json from docker when trying to get ipaddress\nJsonBlob:%s.", err2, jsonBlob)
		return err2, ""
	}
	NetworkSettings := result["NetworkSettings"].(map[string]interface{})
	instance_ip := NetworkSettings["IpAddress"].(string)
	if instance_ip != "" {
		log.Printf("Instance IpAddress: %s", instance_ip)
		return nil, instance_ip
	}
	log.Print("error: Can't get ipaddress...")
	return errors.New("Can't get ipaddress..."), ""
}

// create creates a docker container with base template by default.
// TODO: this template already have a public key, we need to manage to install some way.
func (c *container) create() (err error, instance_id string) {
    runtime, err := docker.NewRuntime()
    if err != nil {
        log.Printf("Error creating docker runtime:%s", err)
        return err, ""
    }
    docker_container, err := runtime.Create(
        &docker.Config{
                Image:  "base-nginx-sshd-key",
                Cmd:    []string{"/usr/sbin/sshd", "-D"},
            },
        )
    if err != nil {
        log.Printf("Error creating docker container: %s", err)
        return err, ""
    }
    log.Printf("Container.State.Running:%b", docker_container.State.Running)
    if err := docker_container.Start(); err != nil {
        log.Printf("Error starting docker container: %s", err)
        return err, ""
    }
    log.Printf("Container.State.Running:%s", docker_container.State.Running)
    log.Printf("container.NetworkSettings.IpAddress:%s", docker_container.NetworkSettings.IpAddress)
    instance_id = docker_container.Id
	log.Printf("docker instance_id=%s", instance_id)
	return err, instance_id
}

// start starts a docker container.
func (c *container) start() error {
	// it isn't necessary to start a docker container after docker run.
	return nil
}

// stop stops a docker container.
func (c *container) stop() error {
	docker, err := config.GetString("docker:binary")
	if err != nil {
		return err
	}
	//TODO: better error handling
	log.Printf("trying to stop instance %s", c.instanceId)
	err, output := runCmd("sudo", docker, "stop", c.instanceId)
	log.Printf("docker stop=%s", output)
	return err
}

// destroy destory a docker container.
func (c *container) destroy() error {
	docker, err := config.GetString("docker:binary")
	if err != nil {
		return err
	}
	//TODO: better error handling
	//TODO: Remove host's nginx route
	log.Printf("trying to destroy instance %s", c.instanceId)
	err, _ = runCmd("sudo", docker, "rm", c.instanceId)
	return err
}

// Copyright 2013 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"bytes"
	"encoding/json"
	"github.com/globocom/tsuru/fs/testing"
	"io"
	"launchpad.net/gocheck"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func (s *S) TestLogin(c *gocheck.C) {
	fsystem = &testing.RecordingFs{FileContent: "old-token"}
	defer func() {
		fsystem = nil
	}()
	expected := "Password: \nSuccessfully logged in!\n"
	reader := strings.NewReader("chico\n")
	context := Context{[]string{"foo@foo.com"}, manager.stdout, manager.stderr, reader}
	client := NewClient(&http.Client{Transport: &transport{msg: `{"token": "sometoken"}`, status: http.StatusOK}}, nil, manager)
	command := login{}
	err := command.Run(&context, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(manager.stdout.(*bytes.Buffer).String(), gocheck.Equals, expected)
	token, err := readToken()
	c.Assert(err, gocheck.IsNil)
	c.Assert(token, gocheck.Equals, "sometoken")
}

func (s *S) TestLoginShouldNotDependOnTsuruTokenFile(c *gocheck.C) {
	rfs := &testing.RecordingFs{}
	f, _ := rfs.Create(joinWithUserDir(".tsuru_target"))
	f.Write([]byte("http://localhost"))
	f.Close()
	fsystem = rfs
	defer func() {
		fsystem = nil
	}()
	expected := "Password: \nSuccessfully logged in!\n"
	reader := strings.NewReader("chico\n")
	context := Context{[]string{"foo@foo.com"}, manager.stdout, manager.stderr, reader}
	client := NewClient(&http.Client{Transport: &transport{msg: `{"token":"anothertoken"}`, status: http.StatusOK}}, nil, manager)
	command := login{}
	err := command.Run(&context, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(manager.stdout.(*bytes.Buffer).String(), gocheck.Equals, expected)
}

func (s *S) TestLoginShouldReturnErrorIfThePasswordIsNotGiven(c *gocheck.C) {
	context := Context{[]string{"foo@foo.com"}, manager.stdout, manager.stderr, strings.NewReader("\n")}
	command := login{}
	err := command.Run(&context, nil)
	c.Assert(err, gocheck.NotNil)
	c.Assert(err, gocheck.ErrorMatches, "^You must provide the password!$")
}

func (s *S) TestLogout(c *gocheck.C) {
	rfs := &testing.RecordingFs{}
	fsystem = rfs
	defer func() {
		fsystem = nil
	}()
	expected := "Successfully logged out!\n"
	context := Context{[]string{}, manager.stdout, manager.stderr, manager.stdin}
	command := logout{}
	err := command.Run(&context, nil)
	c.Assert(err, gocheck.IsNil)
	c.Assert(manager.stdout.(*bytes.Buffer).String(), gocheck.Equals, expected)
	c.Assert(rfs.HasAction("remove "+joinWithUserDir(".tsuru_token")), gocheck.Equals, true)
}

func (s *S) TestLogoutWhenNotLoggedIn(c *gocheck.C) {
	fsystem = &testing.FailureFs{}
	defer func() {
		fsystem = nil
	}()
	context := Context{[]string{}, manager.stdout, manager.stderr, manager.stdin}
	command := logout{}
	err := command.Run(&context, nil)
	c.Assert(err, gocheck.NotNil)
	c.Assert(err.Error(), gocheck.Equals, "You're not logged in!")
}

func (s *S) TestTeamAddUser(c *gocheck.C) {
	expected := `User "andorito" was added to the "cobrateam" team` + "\n"
	context := Context{[]string{"cobrateam", "andorito"}, manager.stdout, manager.stderr, manager.stdin}
	command := teamUserAdd{}
	client := NewClient(&http.Client{Transport: &transport{msg: "", status: http.StatusOK}}, nil, manager)
	err := command.Run(&context, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(manager.stdout.(*bytes.Buffer).String(), gocheck.Equals, expected)
}

func (s *S) TestTeamAddUserInfo(c *gocheck.C) {
	expected := &Info{
		Name:    "team-user-add",
		Usage:   "team-user-add <teamname> <useremail>",
		Desc:    "adds a user to a team.",
		MinArgs: 2,
	}
	c.Assert((&teamUserAdd{}).Info(), gocheck.DeepEquals, expected)
}

func (s *S) TestTeamRemoveUser(c *gocheck.C) {
	expected := `User "andorito" was removed from the "cobrateam" team` + "\n"
	context := Context{[]string{"cobrateam", "andorito"}, manager.stdout, manager.stderr, manager.stdin}
	command := teamUserRemove{}
	client := NewClient(&http.Client{Transport: &transport{msg: "", status: http.StatusOK}}, nil, manager)
	err := command.Run(&context, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(manager.stdout.(*bytes.Buffer).String(), gocheck.Equals, expected)
}

func (s *S) TestTeamRemoveUserInfo(c *gocheck.C) {
	expected := &Info{
		Name:    "team-user-remove",
		Usage:   "team-user-remove <teamname> <useremail>",
		Desc:    "removes a user from a team.",
		MinArgs: 2,
	}
	c.Assert((&teamUserRemove{}).Info(), gocheck.DeepEquals, expected)
}

func (s *S) TestTeamCreate(c *gocheck.C) {
	expected := `Team "core" successfully created!` + "\n"
	context := Context{[]string{"core"}, manager.stdout, manager.stderr, manager.stdin}
	client := NewClient(&http.Client{Transport: &transport{msg: "", status: http.StatusCreated}}, nil, manager)
	command := teamCreate{}
	err := command.Run(&context, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(manager.stdout.(*bytes.Buffer).String(), gocheck.Equals, expected)
}

func (s *S) TestTeamCreateInfo(c *gocheck.C) {
	expected := &Info{
		Name:    "team-create",
		Usage:   "team-create <teamname>",
		Desc:    "creates a new team.",
		MinArgs: 1,
	}
	c.Assert((&teamCreate{}).Info(), gocheck.DeepEquals, expected)
}

func (s *S) TestTeamRemove(c *gocheck.C) {
	var (
		buf    bytes.Buffer
		called bool
	)
	context := Context{
		Args:   []string{"evergrey"},
		Stdout: &buf,
		Stdin:  strings.NewReader("y\n"),
	}
	trans := conditionalTransport{
		transport{
			msg:    "",
			status: http.StatusOK,
		},
		func(req *http.Request) bool {
			called = true
			return req.URL.Path == "/teams/evergrey" && req.Method == "DELETE"
		},
	}
	client := NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := teamRemove{}
	err := command.Run(&context, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(called, gocheck.Equals, true)
	c.Assert(buf.String(), gocheck.Equals, `Are you sure you want to remove team "evergrey"? (y/n) Team "evergrey" successfully removed!`+"\n")
}

func (s *S) TestTeamRemoveWithouConfirmation(c *gocheck.C) {
	var buf bytes.Buffer
	context := Context{
		Args:   []string{"dream-theater"},
		Stdout: &buf,
		Stdin:  strings.NewReader("n\n"),
	}
	command := teamRemove{}
	err := command.Run(&context, nil)
	c.Assert(err, gocheck.IsNil)
	c.Assert(buf.String(), gocheck.Equals, `Are you sure you want to remove team "dream-theater"? (y/n) Abort.`+"\n")
}

func (s *S) TestTeamRemoveFailingRequest(c *gocheck.C) {
	context := Context{
		Args:   []string{"evergrey"},
		Stdout: new(bytes.Buffer),
		Stdin:  strings.NewReader("y\n"),
	}
	client := NewClient(&http.Client{Transport: &transport{msg: "Team evergrey not found.", status: http.StatusNotFound}}, nil, manager)
	command := teamRemove{}
	err := command.Run(&context, client)
	c.Assert(err, gocheck.NotNil)
	c.Assert(err, gocheck.ErrorMatches, "^Team evergrey not found.$")
}

func (s *S) TestTeamRemoveInfo(c *gocheck.C) {
	expected := &Info{
		Name:    "team-remove",
		Usage:   "team-remove <team-name>",
		Desc:    "removes a team from tsuru server.",
		MinArgs: 1,
	}
	c.Assert((&teamRemove{}).Info(), gocheck.DeepEquals, expected)
}

func (s *S) TestTeamRemoveIsACommand(c *gocheck.C) {
	var _ Command = &teamRemove{}
}

func (s *S) TestTeamListRun(c *gocheck.C) {
	var called bool
	trans := &conditionalTransport{
		transport{
			msg:    `[{"name":"timeredbull"},{"name":"cobrateam"}]`,
			status: http.StatusOK,
		},
		func(req *http.Request) bool {
			called = true
			return req.Method == "GET" && req.URL.Path == "/teams"
		},
	}
	expected := `Teams:

  - timeredbull
  - cobrateam
`
	client := NewClient(&http.Client{Transport: trans}, nil, manager)
	err := (&teamList{}).Run(&Context{[]string{}, manager.stdout, manager.stderr, manager.stdin}, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(called, gocheck.Equals, true)
	c.Assert(manager.stdout.(*bytes.Buffer).String(), gocheck.Equals, expected)
}

func (s *S) TestTeamListRunWithNoContent(c *gocheck.C) {
	client := NewClient(&http.Client{Transport: &transport{msg: "", status: http.StatusNoContent}}, nil, manager)
	err := (&teamList{}).Run(&Context{[]string{}, manager.stdout, manager.stderr, manager.stdin}, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(manager.stdout.(*bytes.Buffer).String(), gocheck.Equals, "")
}

func (s *S) TestTeamListInfo(c *gocheck.C) {
	expected := &Info{
		Name:    "team-list",
		Usage:   "team-list",
		Desc:    "List all teams that you are member.",
		MinArgs: 0,
	}
	c.Assert((&teamList{}).Info(), gocheck.DeepEquals, expected)
}

func (s *S) TestTeamListIsACommand(c *gocheck.C) {
	var _ Command = &teamList{}
}

func (s *S) TestUserCreateShouldNotDependOnTsuruTokenFile(c *gocheck.C) {
	rfs := &testing.RecordingFs{}
	f, _ := rfs.Create(joinWithUserDir(".tsuru_target"))
	f.Write([]byte("http://localhost"))
	f.Close()
	fsystem = rfs
	defer func() {
		fsystem = nil
	}()
	expected := "Password: \nConfirm: \n" + `User "foo@foo.com" successfully created!` + "\n"
	reader := strings.NewReader("foo123\nfoo123\n")
	context := Context{[]string{"foo@foo.com"}, manager.stdout, manager.stderr, reader}
	client := NewClient(&http.Client{Transport: &transport{msg: "", status: http.StatusCreated}}, nil, manager)
	command := userCreate{}
	err := command.Run(&context, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(manager.stdout.(*bytes.Buffer).String(), gocheck.Equals, expected)
}

func (s *S) TestUserCreateReturnErrorIfPasswordsDontMatch(c *gocheck.C) {
	reader := strings.NewReader("foo123\nfoo1234\n")
	context := Context{[]string{"foo@foo.com"}, manager.stdout, manager.stderr, reader}
	client := NewClient(&http.Client{Transport: &transport{msg: "", status: http.StatusCreated}}, nil, manager)
	command := userCreate{}
	err := command.Run(&context, client)
	c.Assert(err, gocheck.NotNil)
	c.Assert(err, gocheck.ErrorMatches, "^Passwords didn't match.$")
}

func (s *S) TestUserCreate(c *gocheck.C) {
	expected := "Password: \nConfirm: \n" + `User "foo@foo.com" successfully created!` + "\n"
	context := Context{[]string{"foo@foo.com"}, manager.stdout, manager.stderr, strings.NewReader("foo123\nfoo123\n")}
	client := NewClient(&http.Client{Transport: &transport{msg: "", status: http.StatusCreated}}, nil, manager)
	command := userCreate{}
	err := command.Run(&context, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(manager.stdout.(*bytes.Buffer).String(), gocheck.Equals, expected)
}

func (s *S) TestUserCreateShouldReturnErrorIfThePasswordIsNotGiven(c *gocheck.C) {
	context := Context{[]string{"foo@foo.com"}, manager.stdout, manager.stderr, strings.NewReader("")}
	command := userCreate{}
	err := command.Run(&context, nil)
	c.Assert(err, gocheck.NotNil)
	c.Assert(err, gocheck.ErrorMatches, "^You must provide the password!$")
}

func (s *S) TestUserCreateInfo(c *gocheck.C) {
	expected := &Info{
		Name:    "user-create",
		Usage:   "user-create <email>",
		Desc:    "creates a user.",
		MinArgs: 1,
	}
	c.Assert((&userCreate{}).Info(), gocheck.DeepEquals, expected)
}

func (s *S) TestUserRemove(c *gocheck.C) {
	rfs := &testing.RecordingFs{}
	f, _ := rfs.Create(joinWithUserDir(".tsuru_target"))
	f.Write([]byte("http://tsuru.io"))
	f.Close()
	fsystem = rfs
	defer func() {
		fsystem = nil
	}()
	var (
		buf    bytes.Buffer
		called bool
	)
	context := Context{
		Stdout: &buf,
		Stdin:  strings.NewReader("y\n"),
	}
	trans := conditionalTransport{
		transport{
			msg:    "",
			status: http.StatusOK,
		},
		func(req *http.Request) bool {
			called = true
			return req.Method == "DELETE" && req.URL.Path == "/users"
		},
	}
	client := NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := userRemove{}
	err := command.Run(&context, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(called, gocheck.Equals, true)
	c.Assert(buf.String(), gocheck.Equals, "Are you sure you want to remove your user from tsuru? (y/n) User successfully removed.\n")
	c.Assert(rfs.HasAction("remove "+joinWithUserDir(".tsuru_token")), gocheck.Equals, true)
}

func (s *S) TestUserRemoveWithoutConfirmation(c *gocheck.C) {
	var buf bytes.Buffer
	context := Context{
		Stdout: &buf,
		Stdin:  strings.NewReader("n\n"),
	}
	command := userRemove{}
	err := command.Run(&context, nil)
	c.Assert(err, gocheck.IsNil)
	c.Assert(buf.String(), gocheck.Equals, "Are you sure you want to remove your user from tsuru? (y/n) Abort.\n")
}

func (s *S) TestUserRemoveWithRequestError(c *gocheck.C) {
	client := NewClient(&http.Client{Transport: &transport{msg: "User not found.", status: http.StatusNotFound}}, nil, manager)
	command := userRemove{}
	err := command.Run(&Context{Stdout: new(bytes.Buffer), Stdin: strings.NewReader("y\n")}, client)
	c.Assert(err, gocheck.NotNil)
	c.Assert(err, gocheck.ErrorMatches, "^User not found.$")
}

func (s *S) TestUserRemoveInfo(c *gocheck.C) {
	expected := &Info{
		Name:    "user-remove",
		Usage:   "user-remove",
		Desc:    "removes your user from tsuru server.",
		MinArgs: 0,
	}
	c.Assert((&userRemove{}).Info(), gocheck.DeepEquals, expected)
}

func (s *S) TestUserRemoveIsACommand(c *gocheck.C) {
	var _ Command = &userRemove{}
}

func (s *S) TestChangePassword(c *gocheck.C) {
	var (
		buf    bytes.Buffer
		called bool
		stdin  io.Reader
	)
	stdin = strings.NewReader("gopher\nbbrothers\nbbrothers\n")
	context := Context{
		Stdout: &buf,
		Stdin:  stdin,
	}
	trans := conditionalTransport{
		transport{
			msg:    "",
			status: http.StatusOK,
		},
		func(req *http.Request) bool {
			var got map[string]string
			called = true
			if err := json.NewDecoder(req.Body).Decode(&got); err != nil {
				return false
			}
			cond := got["old"] == "gopher" && got["new"] == "bbrothers"
			return cond && req.Method == "PUT" && req.URL.Path == "/users/password"
		},
	}
	client := NewClient(&http.Client{Transport: &trans}, nil, manager)
	command := changePassword{}
	err := command.Run(&context, client)
	c.Assert(err, gocheck.IsNil)
	c.Assert(called, gocheck.Equals, true)
	expected := "Current password: \nNew password: \nConfirm: \nPassword successfully updated!\n"
	c.Assert(buf.String(), gocheck.Equals, expected)
}

func (s *S) TestChangePasswordWrongConfirmation(c *gocheck.C) {
	var buf bytes.Buffer
	stdin := strings.NewReader("gopher\nblood\nsugar\n")
	context := Context{
		Stdin:  stdin,
		Stdout: &buf,
		Stderr: &buf,
	}
	command := changePassword{}
	err := command.Run(&context, nil)
	c.Assert(err, gocheck.NotNil)
	c.Assert(err.Error(), gocheck.Equals, "New password and password confirmation didn't match.")
}

func (s *S) TestChangePasswordInfo(c *gocheck.C) {
	expected := Info{
		Name:  "change-password",
		Usage: "change-password",
		Desc:  "Change your password.",
	}
	command := changePassword{}
	c.Assert(command.Info(), gocheck.DeepEquals, &expected)
}

func (s *S) TestChangePasswordIsACommand(c *gocheck.C) {
	var _ Command = &changePassword{}
}

func (s *S) TestPasswordFromReaderUsingFile(c *gocheck.C) {
	tmpdir, err := filepath.EvalSymlinks(os.TempDir())
	filename := path.Join(tmpdir, "password-reader.txt")
	c.Assert(err, gocheck.IsNil)
	file, err := os.Create(filename)
	c.Assert(err, gocheck.IsNil)
	defer os.Remove(filename)
	file.WriteString("hello")
	file.Seek(0, 0)
	password, err := passwordFromReader(file)
	c.Assert(err, gocheck.IsNil)
	c.Assert(password, gocheck.Equals, "hello")
}

func (s *S) TestPasswordFromReaderUsingStringsReader(c *gocheck.C) {
	reader := strings.NewReader("abcd\n")
	password, err := passwordFromReader(reader)
	c.Assert(err, gocheck.IsNil)
	c.Assert(password, gocheck.Equals, "abcd")
}

// Copyright 2013 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package app

import (
	"fmt"
	"github.com/globocom/config"
	"github.com/globocom/tsuru/action"
	"github.com/globocom/tsuru/app/bind"
	"labix.org/v2/mgo/bson"
	"launchpad.net/goamz/aws"
	"launchpad.net/goamz/iam"
	"launchpad.net/goamz/s3"
	"launchpad.net/gocheck"
	"strings"
)

func (s *S) TestInsertAppForward(c *gocheck.C) {
	app := App{Name: "conviction", Framework: "evergrey"}
	ctx := action.FWContext{
		Params: []interface{}{app},
	}
	r, err := insertApp.Forward(ctx)
	c.Assert(err, gocheck.IsNil)
	defer s.conn.Apps().Remove(bson.M{"name": app.Name})
	a, ok := r.(*App)
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(a.Name, gocheck.Equals, app.Name)
	c.Assert(a.Framework, gocheck.Equals, app.Framework)
	err = app.Get()
	c.Assert(err, gocheck.IsNil)
}

func (s *S) TestInsertAppForwardAppPointer(c *gocheck.C) {
	app := App{Name: "conviction", Framework: "evergrey"}
	ctx := action.FWContext{
		Params: []interface{}{&app},
	}
	r, err := insertApp.Forward(ctx)
	c.Assert(err, gocheck.IsNil)
	defer s.conn.Apps().Remove(bson.M{"name": app.Name})
	a, ok := r.(*App)
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(a.Name, gocheck.Equals, app.Name)
	c.Assert(a.Framework, gocheck.Equals, app.Framework)
	err = app.Get()
	c.Assert(err, gocheck.IsNil)
}

func (s *S) TestInsertAppForwardInvalidValue(c *gocheck.C) {
	ctx := action.FWContext{
		Params: []interface{}{"hello"},
	}
	r, err := insertApp.Forward(ctx)
	c.Assert(r, gocheck.IsNil)
	c.Assert(err, gocheck.NotNil)
	c.Assert(err.Error(), gocheck.Equals, "First parameter must be App or *App.")
}

func (s *S) TestInsertAppDuplication(c *gocheck.C) {
	app := App{Name: "come", Framework: "gotthard"}
	err := s.conn.Apps().Insert(app)
	c.Assert(err, gocheck.IsNil)
	defer s.conn.Apps().Remove(bson.M{"name": app.Name})
	ctx := action.FWContext{
		Params: []interface{}{&app},
	}
	r, err := insertApp.Forward(ctx)
	c.Assert(r, gocheck.IsNil)
	c.Assert(err, gocheck.NotNil)
	c.Assert(err.Error(), gocheck.Equals, "there is already an app with this name.")
}

func (s *S) TestInsertAppBackward(c *gocheck.C) {
	app := App{Name: "conviction", Framework: "evergrey"}
	ctx := action.BWContext{
		Params:   []interface{}{app},
		FWResult: &app,
	}
	err := s.conn.Apps().Insert(app)
	c.Assert(err, gocheck.IsNil)
	defer s.conn.Apps().Remove(bson.M{"name": app.Name}) // sanity
	insertApp.Backward(ctx)
	n, err := s.conn.Apps().Find(bson.M{"name": app.Name}).Count()
	c.Assert(err, gocheck.IsNil)
	c.Assert(n, gocheck.Equals, 0)
}

func (s *S) TestInsertAppMinimumParams(c *gocheck.C) {
	c.Assert(insertApp.MinParams, gocheck.Equals, 1)
}

func (s *S) TestCreateIAMUserForward(c *gocheck.C) {
	auth := aws.Auth{AccessKey: "access", SecretKey: "s3cr3t"}
	region := aws.Region{IAMEndpoint: s.t.IamServer.URL()}
	iamClient := iam.New(auth, region)
	app := App{Name: "trapped"}
	ctx := action.FWContext{Params: []interface{}{&app}, Previous: &app}
	result, err := createIAMUserAction.Forward(ctx)
	c.Assert(err, gocheck.IsNil)
	defer iamClient.DeleteUser(app.Name)
	u, ok := result.(*iam.User)
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(u.Name, gocheck.Equals, app.Name)
}

func (s *S) TestCreateIAMUserBackward(c *gocheck.C) {
	auth := aws.Auth{AccessKey: "access", SecretKey: "s3cr3t"}
	region := aws.Region{IAMEndpoint: s.t.IamServer.URL()}
	iamClient := iam.New(auth, region)
	app := App{Name: "escape"}
	user, err := createIAMUser(app.Name)
	c.Assert(err, gocheck.IsNil)
	defer iamClient.DeleteUser(user.Name)
	ctx := action.BWContext{Params: []interface{}{&app}, FWResult: user}
	createIAMUserAction.Backward(ctx)
	_, err = iamClient.GetUser(user.Name)
	c.Assert(err, gocheck.NotNil)
}

func (s *S) TestCreateIAMUserMinParams(c *gocheck.C) {
	c.Assert(createIAMUserAction.MinParams, gocheck.Equals, 1)
}

func (s *S) TestCreateIAMAccessKeyForward(c *gocheck.C) {
	auth := aws.Auth{AccessKey: "access", SecretKey: "s3cr3t"}
	region := aws.Region{IAMEndpoint: s.t.IamServer.URL()}
	iamClient := iam.New(auth, region)
	resp, err := iamClient.CreateUser("puppets", "/")
	c.Assert(err, gocheck.IsNil)
	defer iamClient.DeleteUser(resp.User.Name)
	ctx := action.FWContext{Params: []interface{}{nil}, Previous: &resp.User}
	result, err := createIAMAccessKeyAction.Forward(ctx)
	c.Assert(err, gocheck.IsNil)
	ak, ok := result.(*iam.AccessKey)
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(ak.UserName, gocheck.Equals, resp.User.Name)
	c.Assert(ak.Id, gocheck.Not(gocheck.Equals), "")
	c.Assert(ak.Secret, gocheck.Equals, "")
	defer iamClient.DeleteAccessKey(ak.Id, ak.UserName)
}

func (s *S) TestCreateIAMAccessKeyBackward(c *gocheck.C) {
	auth := aws.Auth{AccessKey: "access", SecretKey: "s3cr3t"}
	region := aws.Region{IAMEndpoint: s.t.IamServer.URL()}
	iamClient := iam.New(auth, region)
	resp, err := iamClient.CreateUser("myuser", "/")
	c.Assert(err, gocheck.IsNil)
	defer iamClient.DeleteUser(resp.User.Name)
	kresp, err := iamClient.CreateAccessKey(resp.User.Name)
	c.Assert(err, gocheck.IsNil)
	defer iamClient.DeleteAccessKey(kresp.AccessKey.Id, resp.User.Name)
	ctx := action.BWContext{Params: []interface{}{nil}, FWResult: &kresp.AccessKey}
	createIAMAccessKeyAction.Backward(ctx)
	akResp, err := iamClient.AccessKeys(resp.User.Name)
	c.Assert(err, gocheck.IsNil)
	c.Assert(akResp.AccessKeys, gocheck.HasLen, 0)
}

func (s *S) TestCreateIAMMinParams(c *gocheck.C) {
	c.Assert(createIAMAccessKeyAction.MinParams, gocheck.Equals, 1)
}

func (s *S) TestCreateBucketForward(c *gocheck.C) {
	auth := aws.Auth{AccessKey: "access", SecretKey: "s3cr3t"}
	region := aws.Region{S3Endpoint: s.t.S3Server.URL()}
	s3Client := s3.New(auth, region)
	app := App{Name: "leper"}
	ctx := action.FWContext{
		Params:   []interface{}{&app},
		Previous: &iam.AccessKey{Id: "access", Secret: "s3cr3t"},
	}
	result, err := createBucketAction.Forward(ctx)
	c.Assert(err, gocheck.IsNil)
	env, ok := result.(*s3Env)
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(env.AccessKey, gocheck.Equals, "access")
	c.Assert(env.SecretKey, gocheck.Equals, "s3cr3t")
	c.Assert(env.endpoint, gocheck.Equals, s.t.S3Server.URL())
	c.Assert(env.locationConstraint, gocheck.Equals, true)
	defer s3Client.Bucket(env.bucket).DelBucket()
	_, err = s3Client.Bucket(env.bucket).List("", "/", "", 100)
	c.Assert(err, gocheck.IsNil)
}

func (s *S) TestCreateBucketBackward(c *gocheck.C) {
	patchRandomReader()
	defer unpatchRandomReader()
	auth := aws.Auth{AccessKey: "access", SecretKey: "s3cr3t"}
	region := aws.Region{
		Name:                 "myregion",
		S3Endpoint:           s.t.S3Server.URL(),
		S3LocationConstraint: true,
		S3LowercaseBucket:    true,
	}
	s3Client := s3.New(auth, region)
	app := App{Name: "leper"}
	err := s3Client.Bucket(app.Name).PutBucket(s3.BucketOwnerFull)
	c.Assert(err, gocheck.IsNil)
	env := s3Env{
		Auth:               aws.Auth{AccessKey: "access", SecretKey: "s3cr3t"},
		bucket:             app.Name,
		endpoint:           s.t.S3Server.URL(),
		locationConstraint: true,
	}
	ctx := action.BWContext{Params: []interface{}{&app}, FWResult: &env}
	createBucketAction.Backward(ctx)
	_, err = s3Client.Bucket(app.Name).List("", "/", "", 100)
	c.Assert(err, gocheck.NotNil)
}

func (s *S) TestCreateBucketMinParams(c *gocheck.C) {
	c.Assert(createBucketAction.MinParams, gocheck.Equals, 1)
}

func (s *S) TestCreateUserPolicyForward(c *gocheck.C) {
	auth := aws.Auth{AccessKey: "access", SecretKey: "s3cr3t"}
	region := aws.Region{IAMEndpoint: s.t.IamServer.URL()}
	iamClient := iam.New(auth, region)
	resp, err := iamClient.CreateUser("blackened", "/")
	c.Assert(err, gocheck.IsNil)
	defer iamClient.DeleteUser(resp.User.Name)
	app := App{Name: resp.User.Name}
	env := s3Env{
		Auth:               aws.Auth{AccessKey: "access", SecretKey: "s3cr3t"},
		bucket:             app.Name,
		endpoint:           s.t.S3Server.URL(),
		locationConstraint: true,
	}
	ctx := action.FWContext{Params: []interface{}{&app}, Previous: &env}
	result, err := createUserPolicyAction.Forward(ctx)
	c.Assert(err, gocheck.IsNil)
	e, ok := result.(*s3Env)
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(e, gocheck.Equals, &env)
	_, err = iamClient.GetUserPolicy(resp.User.Name, "app-blackened-bucket")
	c.Assert(err, gocheck.IsNil)
}

func (s *S) TestCreateUserPolicyBackward(c *gocheck.C) {
	auth := aws.Auth{AccessKey: "access", SecretKey: "s3cr3t"}
	region := aws.Region{IAMEndpoint: s.t.IamServer.URL()}
	iamClient := iam.New(auth, region)
	resp, err := iamClient.CreateUser("blackened", "/")
	c.Assert(err, gocheck.IsNil)
	defer iamClient.DeleteUser(resp.User.Name)
	app := App{Name: resp.User.Name}
	env := s3Env{
		Auth:               aws.Auth{AccessKey: "access", SecretKey: "s3cr3t"},
		bucket:             app.Name,
		endpoint:           s.t.S3Server.URL(),
		locationConstraint: true,
	}
	_, err = iamClient.PutUserPolicy(resp.User.Name, "app-blackened-bucket", "null")
	c.Assert(err, gocheck.IsNil)
	ctx := action.BWContext{Params: []interface{}{&app}, FWResult: &env}
	createUserPolicyAction.Backward(ctx)
	_, err = iamClient.GetUserPolicy(resp.User.Name, "app-blackened-bucket")
	c.Assert(err, gocheck.NotNil)
}

func (s *S) TestCreateUsePolicyMinParams(c *gocheck.C) {
	c.Assert(createUserPolicyAction.MinParams, gocheck.Equals, 1)
}

func (s *S) TestExportEnvironmentsForward(c *gocheck.C) {
	expectedHost := "localhost"
	config.Set("host", expectedHost)
	app := App{Name: "mist", Framework: "opeth"}
	err := s.conn.Apps().Insert(app)
	c.Assert(err, gocheck.IsNil)
	defer s.conn.Apps().Remove(bson.M{"name": app.Name})
	env := s3Env{
		Auth:               aws.Auth{AccessKey: "access", SecretKey: "s3cr3t"},
		bucket:             app.Name + "-bucket",
		endpoint:           s.t.S3Server.URL(),
		locationConstraint: true,
	}
	ctx := action.FWContext{Params: []interface{}{&app}, Previous: &env}
	result, err := exportEnvironmentsAction.Forward(ctx)
	c.Assert(err, gocheck.IsNil)
	c.Assert(result, gocheck.Equals, &env)
	err = app.Get()
	c.Assert(err, gocheck.IsNil)
	appEnv := app.InstanceEnv(s3InstanceName)
	c.Assert(appEnv["TSURU_S3_ENDPOINT"].Value, gocheck.Equals, env.endpoint)
	c.Assert(appEnv["TSURU_S3_ENDPOINT"].Public, gocheck.Equals, false)
	c.Assert(appEnv["TSURU_S3_LOCATIONCONSTRAINT"].Value, gocheck.Equals, "true")
	c.Assert(appEnv["TSURU_S3_LOCATIONCONSTRAINT"].Public, gocheck.Equals, false)
	c.Assert(appEnv["TSURU_S3_ACCESS_KEY_ID"].Value, gocheck.Equals, env.AccessKey)
	c.Assert(appEnv["TSURU_S3_ACCESS_KEY_ID"].Public, gocheck.Equals, false)
	c.Assert(appEnv["TSURU_S3_SECRET_KEY"].Value, gocheck.Equals, env.SecretKey)
	c.Assert(appEnv["TSURU_S3_SECRET_KEY"].Public, gocheck.Equals, false)
	c.Assert(appEnv["TSURU_S3_BUCKET"].Value, gocheck.Equals, env.bucket)
	c.Assert(appEnv["TSURU_S3_BUCKET"].Public, gocheck.Equals, false)
	appEnv = app.InstanceEnv("")
	c.Assert(appEnv["TSURU_APPNAME"].Value, gocheck.Equals, app.Name)
	c.Assert(appEnv["TSURU_APPNAME"].Public, gocheck.Equals, false)
	c.Assert(appEnv["TSURU_HOST"].Value, gocheck.Equals, expectedHost)
	c.Assert(appEnv["TSURU_HOST"].Public, gocheck.Equals, false)
	message, err := aqueue().Get(2e9)
	c.Assert(err, gocheck.IsNil)
	defer message.Delete()
	c.Assert(message.Action, gocheck.Equals, regenerateApprc)
	c.Assert(message.Args, gocheck.DeepEquals, []string{app.Name})
}

func (s *S) TestExportEnvironmentsForwardWithoutS3Env(c *gocheck.C) {
	expectedHost := "localhost"
	config.Set("host", expectedHost)
	app := App{Name: "mist", Framework: "opeth"}
	err := s.conn.Apps().Insert(app)
	c.Assert(err, gocheck.IsNil)
	defer s.conn.Apps().Remove(bson.M{"name": app.Name})
	ctx := action.FWContext{Params: []interface{}{&app}, Previous: &app}
	result, err := exportEnvironmentsAction.Forward(ctx)
	c.Assert(err, gocheck.IsNil)
	c.Assert(result, gocheck.Equals, &app)
	err = app.Get()
	c.Assert(err, gocheck.IsNil)
	appEnv := app.InstanceEnv(s3InstanceName)
	c.Assert(appEnv, gocheck.DeepEquals, map[string]bind.EnvVar{})
	appEnv = app.InstanceEnv("")
	c.Assert(appEnv["TSURU_APPNAME"].Value, gocheck.Equals, app.Name)
	c.Assert(appEnv["TSURU_APPNAME"].Public, gocheck.Equals, false)
	c.Assert(appEnv["TSURU_HOST"].Value, gocheck.Equals, expectedHost)
	c.Assert(appEnv["TSURU_HOST"].Public, gocheck.Equals, false)
}

func (s *S) TestExportEnvironmentsBackward(c *gocheck.C) {
	envNames := []string{
		"TSURU_S3_ACCESS_KEY_ID", "TSURU_S3_SECRET_KEY",
		"TSURU_APPNAME", "TSURU_HOST", "TSURU_S3_ENDPOINT",
		"TSURU_S3_LOCATIONCONSTRAINT", "TSURU_S3_BUCKET",
	}
	app := App{Name: "moon", Framework: "opeth", Env: make(map[string]bind.EnvVar)}
	for _, name := range envNames {
		envVar := bind.EnvVar{Name: name, Value: name, Public: false}
		if strings.HasPrefix(name, "TSURU_S3_") {
			envVar.InstanceName = s3InstanceName
		}
		app.Env[name] = envVar
	}
	err := s.conn.Apps().Insert(app)
	c.Assert(err, gocheck.IsNil)
	defer s.conn.Apps().Remove(bson.M{"name": app.Name})
	ctx := action.BWContext{Params: []interface{}{&app}}
	exportEnvironmentsAction.Backward(ctx)
	copy := app
	err = copy.Get()
	c.Assert(err, gocheck.IsNil)
	for _, name := range envNames {
		if _, ok := copy.Env[name]; ok {
			c.Errorf("Variable %q should be unexported, but it's still exported.", name)
		}
	}
}

func (s *S) TestExportEnvironmentsMinParams(c *gocheck.C) {
	c.Assert(exportEnvironmentsAction.MinParams, gocheck.Equals, 1)
}

func (s *S) TestCreateRepositoryForward(c *gocheck.C) {
	h := testHandler{}
	ts := s.t.StartGandalfTestServer(&h)
	defer ts.Close()
	app := App{Name: "someapp", Teams: []string{s.team.Name}}
	ctx := action.FWContext{Params: []interface{}{app}}
	result, err := createRepository.Forward(ctx)
	a, ok := result.(*App)
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(a.Name, gocheck.Equals, app.Name)
	c.Assert(err, gocheck.IsNil)
	c.Assert(h.url[0], gocheck.Equals, "/repository")
	c.Assert(h.method[0], gocheck.Equals, "POST")
	expected := fmt.Sprintf(`{"name":"someapp","users":["%s"],"ispublic":false}`, s.user.Email)
	c.Assert(string(h.body[0]), gocheck.Equals, expected)
}

func (s *S) TestCreateRepositoryForwardAppPointer(c *gocheck.C) {
	h := testHandler{}
	ts := s.t.StartGandalfTestServer(&h)
	defer ts.Close()
	app := App{Name: "someapp", Teams: []string{s.team.Name}}
	ctx := action.FWContext{Params: []interface{}{&app}}
	result, err := createRepository.Forward(ctx)
	a, ok := result.(*App)
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(a.Name, gocheck.Equals, app.Name)
	c.Assert(err, gocheck.IsNil)
	c.Assert(h.url[0], gocheck.Equals, "/repository")
	c.Assert(h.method[0], gocheck.Equals, "POST")
	expected := fmt.Sprintf(`{"name":"someapp","users":["%s"],"ispublic":false}`, s.user.Email)
	c.Assert(string(h.body[0]), gocheck.Equals, expected)
}

func (s *S) TestCreateRepositoryForwardInvalidType(c *gocheck.C) {
	ctx := action.FWContext{Params: []interface{}{"something"}}
	_, err := createRepository.Forward(ctx)
	c.Assert(err, gocheck.NotNil)
	c.Assert(err.Error(), gocheck.Equals, "First parameter must be App or *App.")
}

func (s *S) TestCreateRepositoryBackward(c *gocheck.C) {
	h := testHandler{}
	ts := s.t.StartGandalfTestServer(&h)
	defer ts.Close()
	app := App{Name: "someapp"}
	ctx := action.BWContext{FWResult: &app, Params: []interface{}{app}}
	createRepository.Backward(ctx)
	c.Assert(h.url[0], gocheck.Equals, "/repository/someapp")
	c.Assert(h.method[0], gocheck.Equals, "DELETE")
	c.Assert(string(h.body[0]), gocheck.Equals, "null")
}

func (s *S) TestCreateRepositoryMinParams(c *gocheck.C) {
	c.Assert(createRepository.MinParams, gocheck.Equals, 1)
}

func (s *S) TestProvisionAppForward(c *gocheck.C) {
	app := App{
		Name:      "earthshine",
		Framework: "django",
		Units:     []Unit{{Machine: 3}},
	}
	err := s.conn.Apps().Insert(app)
	c.Assert(err, gocheck.IsNil)
	defer s.conn.Apps().Remove(bson.M{"name": app.Name})
	ctx := action.FWContext{Params: []interface{}{app, 4}}
	result, err := provisionApp.Forward(ctx)
	defer s.provisioner.Destroy(&app)
	c.Assert(err, gocheck.IsNil)
	a, ok := result.(*App)
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(a.Name, gocheck.Equals, app.Name)
	index := s.provisioner.FindApp(&app)
	c.Assert(index, gocheck.Equals, 0)
}

func (s *S) TestProvisionAppForwardAppPointer(c *gocheck.C) {
	app := App{
		Name:      "earthshine",
		Framework: "django",
		Units:     []Unit{{Machine: 3}},
	}
	err := s.conn.Apps().Insert(app)
	c.Assert(err, gocheck.IsNil)
	defer s.conn.Apps().Remove(bson.M{"name": app.Name})
	ctx := action.FWContext{Params: []interface{}{&app, 4}}
	result, err := provisionApp.Forward(ctx)
	defer s.provisioner.Destroy(&app)
	c.Assert(err, gocheck.IsNil)
	a, ok := result.(*App)
	c.Assert(ok, gocheck.Equals, true)
	c.Assert(a.Name, gocheck.Equals, app.Name)
	index := s.provisioner.FindApp(&app)
	c.Assert(index, gocheck.Equals, 0)
}

func (s *S) TestProvisionAppForwardInvalidApp(c *gocheck.C) {
	ctx := action.FWContext{Params: []interface{}{"something", 1}}
	_, err := provisionApp.Forward(ctx)
	c.Assert(err, gocheck.NotNil)
}

func (s *S) TestProvisionAppBackward(c *gocheck.C) {
	app := App{
		Name:      "earthshine",
		Framework: "django",
		Units:     []Unit{{Machine: 3}},
	}
	err := s.conn.Apps().Insert(app)
	c.Assert(err, gocheck.IsNil)
	defer s.conn.Apps().Remove(bson.M{"name": app.Name})
	fwctx := action.FWContext{Params: []interface{}{&app, 4}}
	result, err := provisionApp.Forward(fwctx)
	c.Assert(err, gocheck.IsNil)
	bwctx := action.BWContext{Params: []interface{}{&app, 4}, FWResult: result}
	provisionApp.Backward(bwctx)
	index := s.provisioner.FindApp(&app)
	c.Assert(index, gocheck.Equals, -1)
}

func (s *S) TestProvisionAppMinParams(c *gocheck.C) {
	c.Assert(provisionApp.MinParams, gocheck.Equals, 2)
}

func (s *S) TestProvisionAddUnitsForward(c *gocheck.C) {
	app := App{
		Name:      "castle",
		Framework: "heavens",
		Units:     []Unit{{Machine: 2}},
	}
	err := s.conn.Apps().Insert(app)
	c.Assert(err, gocheck.IsNil)
	defer s.conn.Apps().Remove(bson.M{"name": app.Name})
	previous, err := provisionApp.Forward(action.FWContext{Params: []interface{}{&app, 4}})
	c.Assert(err, gocheck.IsNil)
	defer provisionApp.Backward(action.BWContext{Params: []interface{}{&app, 4}, FWResult: previous})
	ctx := action.FWContext{Params: []interface{}{&app, 4}, Previous: previous}
	result, err := provisionAddUnits.Forward(ctx)
	c.Assert(err, gocheck.IsNil)
	c.Assert(result, gocheck.IsNil)
	units := s.provisioner.GetUnits(&app)
	c.Assert(units, gocheck.HasLen, 4)
}

func (s *S) TestProvisionAddUnitsBackward(c *gocheck.C) {
	c.Assert(provisionAddUnits.Backward, gocheck.IsNil)
}

func (s *S) TestProvisionAddUnitsMinParams(c *gocheck.C) {
	c.Assert(provisionAddUnits.MinParams, gocheck.Equals, 2)
}

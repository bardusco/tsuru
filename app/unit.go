// Copyright 2013 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package app

import (
	"github.com/globocom/tsuru/provision"
)

// Unit is the smaller bit in tsuru. Each app is composed of one or more units.
//
// The unit is equivalent to a machine. How the machine is actually represented
// (baremetal, virtual machine, jails, containers, etc.) is up to the
// provisioner.
type Unit struct {
	Name       string
	Type       string
	Machine    int
	InstanceId string
	Ip         string
	State      string
	app        *App
}

func (u *Unit) GetName() string {
	return u.Name
}

func (u *Unit) GetMachine() int {
	return u.Machine
}

func (u *Unit) GetIp() string {
	return u.Ip
}

func (u *Unit) GetStatus() provision.Status {
	return provision.Status(u.State)
}

func (u *Unit) GetInstanceId() string {
	return u.InstanceId
}

// UnitSlice attaches the methods of sort.Interface to []Unit, sorting in increasing order.
type UnitSlice []Unit

func (u UnitSlice) Len() int {
	return len(u)
}

func (u UnitSlice) Less(i, j int) bool {
	weight := map[string]int{
		string(provision.StatusError):      0,
		string(provision.StatusDown):       1,
		string(provision.StatusPending):    2,
		string(provision.StatusCreating):   3,
		string(provision.StatusInstalling): 4,
		string(provision.StatusStarted):    5,
	}
	return weight[u[i].State] < weight[u[j].State]
}

func (u UnitSlice) Swap(i, j int) {
	u[i], u[j] = u[j], u[i]
}

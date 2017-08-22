// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package utils_test

import (
	"applatix.io/axops/tool"
	"applatix.io/axops/utils"
	"gopkg.in/check.v1"
	"os"
	"testing"
)

type S struct{}

func Test(t *testing.T) { check.TestingT(t) }

var _ = check.Suite(&S{})

func (s *S) TestWriteReadFile(c *check.C) {
	defer os.RemoveAll("cert.crt")
	cert :=
		`-----BEGIN CERTIFICATE-----
CERTIFICATE
-----END CERTIFICATE-----`

	err := utils.WriteToFile(cert, "cert.crt")
	c.Assert(err, check.IsNil)

	copy, err := utils.ReadFromFile("cert.crt")
	c.Assert(err, check.IsNil)
	c.Assert(copy, check.Equals, cert)
}

func (s *S) TestGetUserEmail(c *check.C) {
	email := utils.GetUserEmail("User <user@example.com>")
	c.Assert(email, check.Equals, "user@example.com")

	email = utils.GetUserEmail("user@example.com")
	c.Assert(email, check.Equals, "user@example.com")

	email = utils.GetUserEmail("user@ex")
	c.Assert(email, check.Equals, "")
}

func (s *S) TestParseRepoURL(c *check.C) {
	var repoURLs = []string{
		"ssh://git@github.com/github/git.git",
		"git@github.com:github/git.git",
		"ssh://user@other.host.com/~/github/git.git",
		"https://github.com/github/git.git",
		"git://github.com/github/git.git",
	}
	for _, url := range repoURLs {
		owner, name := utils.ParseRepoURL(url)
		c.Assert(owner, check.Equals, "github")
		c.Assert(name, check.Equals, "git")
	}
}

func (s *S) TestGenerateSelfSignedCert(c *check.C) {
	crt, key := utils.GenerateSelfSignedCert()
	err := tool.ValidateCertKeyPair(crt, key)
	c.Assert(err, check.IsNil)
}

func (s *S) TestGetParamsFromString(c *check.C) {
	seq := "%%"
	str := ""
	params := utils.GetParamsFromString(str, seq)
	c.Assert(params, check.DeepEquals, []string{})

	str = "aaaaa"
	params = utils.GetParamsFromString(str, seq)
	c.Assert(params, check.DeepEquals, []string{})

	str = "%%CMD%%"
	params = utils.GetParamsFromString(str, seq)
	c.Assert(params, check.DeepEquals, []string{"CMD"})

	str = "a%%CMD%%"
	params = utils.GetParamsFromString(str, seq)
	c.Assert(params, check.DeepEquals, []string{"CMD"})

	str = "a%%CMD%%b"
	params = utils.GetParamsFromString(str, seq)
	c.Assert(params, check.DeepEquals, []string{"CMD"})

	str = "%%a%%%%b%%"
	params = utils.GetParamsFromString(str, seq)
	c.Assert(params, check.DeepEquals, []string{"a", "b"})

	str = "c%%a%%%%b%%"
	params = utils.GetParamsFromString(str, seq)
	c.Assert(params, check.DeepEquals, []string{"a", "b"})

	str = "%%a%%c%%b%%"
	params = utils.GetParamsFromString(str, seq)
	c.Assert(params, check.DeepEquals, []string{"a", "b"})

	str = "%%a%%%%b%%c"
	params = utils.GetParamsFromString(str, seq)
	c.Assert(params, check.DeepEquals, []string{"a", "b"})

	str = "c%%a%%c%%b%%c"
	params = utils.GetParamsFromString(str, seq)
	c.Assert(params, check.DeepEquals, []string{"a", "b"})

	str = "c%%a%%c%%b%%"
	params = utils.GetParamsFromString(str, seq)
	c.Assert(params, check.DeepEquals, []string{"a", "b"})

	str = "   c%%a%%   c%%b%%"
	params = utils.GetParamsFromString(str, seq)
	c.Assert(params, check.DeepEquals, []string{"a", "b"})

	str = "   c%%a%%   c%%b%%    "
	params = utils.GetParamsFromString(str, seq)
	c.Assert(params, check.DeepEquals, []string{"a", "b"})

	str = "axscm clone %%repo%% /src --commit %%commit%%"
	params = utils.GetParamsFromString(str, seq)
	c.Assert(params, check.DeepEquals, []string{"repo", "commit"})

	str = "/src/build/build_saas.py -bl -n %%namespace%% -v %%version%% -s %%service%%"
	params = utils.GetParamsFromString(str, seq)
	c.Assert(params, check.DeepEquals, []string{"namespace", "version", "service"})
}

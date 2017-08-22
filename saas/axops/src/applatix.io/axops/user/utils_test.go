package user

import (
	"gopkg.in/check.v1"
	"testing"
)

type S struct{}

func Test(t *testing.T) { check.TestingT(t) }

var _ = check.Suite(&S{})

func (s *S) TestPasswordTooShort(c *check.C) {
	password := "1234567"
	err := checkPasswordStrength(password)
	c.Assert(err, check.NotNil)
}

func (s *S) TestPasswordNoUpperCase(c *check.C) {
	password := "aaaaa@12345"
	err := checkPasswordStrength(password)
	c.Assert(err, check.NotNil)
}

func (s *S) TestPasswordNoLowerCase(c *check.C) {
	password := "AAAAA@12345"
	err := checkPasswordStrength(password)
	c.Assert(err, check.NotNil)
}

func (s *S) TestPasswordNoSpeical(c *check.C) {
	password := "Aaaaaa12345"
	err := checkPasswordStrength(password)
	c.Assert(err, check.NotNil)
}

func (s *S) TestPasswordGood(c *check.C) {
	password := "Abcd-12345"
	err := checkPasswordStrength(password)
	c.Assert(err, check.IsNil)

	password = "Abcd!12345"
	err = checkPasswordStrength(password)
	c.Assert(err, check.IsNil)

	password = "Abcd!12345"
	err = checkPasswordStrength(password)
	c.Assert(err, check.IsNil)

	password = "Abcd@12345"
	err = checkPasswordStrength(password)
	c.Assert(err, check.IsNil)

	password = "Abcd#12345"
	err = checkPasswordStrength(password)
	c.Assert(err, check.IsNil)

	password = "Abcd$12345"
	err = checkPasswordStrength(password)
	c.Assert(err, check.IsNil)

	password = "Abcd%12345"
	err = checkPasswordStrength(password)
	c.Assert(err, check.IsNil)

	password = "Abcd^12345"
	err = checkPasswordStrength(password)
	c.Assert(err, check.IsNil)

	password = "Abcd&12345"
	err = checkPasswordStrength(password)
	c.Assert(err, check.IsNil)

	password = "Abcd*12345"
	err = checkPasswordStrength(password)
	c.Assert(err, check.IsNil)

	password = "Abcd(12345"
	err = checkPasswordStrength(password)
	c.Assert(err, check.IsNil)

	password = "Abcd)12345"
	err = checkPasswordStrength(password)
	c.Assert(err, check.IsNil)

	password = "Abcd-12345"
	err = checkPasswordStrength(password)
	c.Assert(err, check.IsNil)

	password = "Abcd=12345"
	err = checkPasswordStrength(password)
	c.Assert(err, check.IsNil)

	password = "Abcd+12345"
	err = checkPasswordStrength(password)
	c.Assert(err, check.IsNil)

	password = "Abcd[12345"
	err = checkPasswordStrength(password)
	c.Assert(err, check.IsNil)

	password = "Abcd]12345"
	err = checkPasswordStrength(password)
	c.Assert(err, check.IsNil)

	password = "Abcd{12345"
	err = checkPasswordStrength(password)
	c.Assert(err, check.IsNil)

	password = "Abcd}12345"
	err = checkPasswordStrength(password)
	c.Assert(err, check.IsNil)

	password = "Abcd:12345"
	err = checkPasswordStrength(password)
	c.Assert(err, check.IsNil)

	password = "Abcd;12345"
	err = checkPasswordStrength(password)
	c.Assert(err, check.IsNil)

	password = "Abcd\"12345"
	err = checkPasswordStrength(password)
	c.Assert(err, check.IsNil)

	password = "Abcd'12345"
	err = checkPasswordStrength(password)
	c.Assert(err, check.IsNil)

	password = "Abcd,12345"
	err = checkPasswordStrength(password)
	c.Assert(err, check.IsNil)

	password = "Abcd.12345"
	err = checkPasswordStrength(password)
	c.Assert(err, check.IsNil)

	password = "Abcd?12345"
	err = checkPasswordStrength(password)
	c.Assert(err, check.IsNil)

	password = "Abcd/12345"
	err = checkPasswordStrength(password)
	c.Assert(err, check.IsNil)

	password = "Abcd<12345"
	err = checkPasswordStrength(password)
	c.Assert(err, check.IsNil)

	password = "Abcd>12345"
	err = checkPasswordStrength(password)
	c.Assert(err, check.IsNil)

	password = "Abcd{12345"
	err = checkPasswordStrength(password)
	c.Assert(err, check.IsNil)

	password = "Abcd}12345"
	err = checkPasswordStrength(password)
	c.Assert(err, check.IsNil)

	password = "Abcd\\12345"
	err = checkPasswordStrength(password)
	c.Assert(err, check.IsNil)

	password = "Abcd~12345"
	err = checkPasswordStrength(password)
	c.Assert(err, check.IsNil)

	password = "Abcd|12345"
	err = checkPasswordStrength(password)
	c.Assert(err, check.IsNil)
}

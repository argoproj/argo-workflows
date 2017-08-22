package rediscl_test

import (
	"applatix.io/test"
	"gopkg.in/check.v1"
	"time"
)

func (s *S) TestRedisCRUD(c *check.C) {
	key := test.RandStr()
	val1 := test.RandStr()
	val2 := test.RandStr()

	err := client.Set(key, val1)
	c.Assert(err, check.IsNil)

	result, err := client.GetString(key)
	c.Assert(err, check.IsNil)
	c.Assert(result, check.Equals, val1)

	err = client.Set(key, val2)
	c.Assert(err, check.IsNil)

	result, err = client.GetString(key)
	c.Assert(err, check.IsNil)
	c.Assert(result, check.Equals, val2)

	err = client.Del(key)
	c.Assert(err, check.IsNil)

	result, err = client.GetString(key)
	c.Assert(err, check.IsNil)
	c.Assert(result, check.NotNil)
	c.Assert(result, check.Equals, "")
}

func (s *S) TestRedisCRUDTLL(c *check.C) {
	key := test.RandStr()
	val1 := test.RandStr()

	err := client.SetWithTTL(key, val1, time.Second)
	c.Assert(err, check.IsNil)

	result, err := client.GetString(key)
	c.Assert(err, check.IsNil)
	c.Assert(result, check.Equals, val1)

	time.Sleep(time.Second)

	result, err = client.GetString(key)
	c.Assert(err, check.IsNil)
	c.Assert(result, check.NotNil)
	c.Assert(result, check.Equals, "")
}

type Obj struct {
	Name string `json:"name"`
}

func (s *S) TestRedisObjCRUD(c *check.C) {
	key := test.RandStr()
	val1 := Obj{test.RandStr()}
	val2 := Obj{test.RandStr()}

	err := client.SetObj(key, val1)
	c.Assert(err, check.IsNil)

	obj := Obj{}
	err = client.GetObj(key, &obj)
	c.Assert(err, check.IsNil)
	c.Assert(val1.Name, check.Equals, obj.Name)

	err = client.SetObj(key, val2)
	c.Assert(err, check.IsNil)

	obj = Obj{}
	err = client.GetObj(key, &obj)
	c.Assert(err, check.IsNil)
	c.Assert(val2.Name, check.Equals, obj.Name)

	err = client.Del(key)
	c.Assert(err, check.IsNil)

	obj = Obj{}
	err = client.GetObj(key, &obj)
	c.Assert(err, check.NotNil)
}

func (s *S) TestRedisObjCRUDTLL(c *check.C) {
	key := test.RandStr()
	val1 := Obj{test.RandStr()}

	err := client.SetObjWithTTL(key, val1, time.Second)
	c.Assert(err, check.IsNil)

	obj := Obj{}
	err = client.GetObj(key, &obj)
	c.Assert(err, check.IsNil)
	c.Assert(val1.Name, check.Equals, obj.Name)

	time.Sleep(time.Second)

	obj = Obj{}
	err = client.GetObj(key, &obj)
	c.Assert(err, check.NotNil)
}

func (s *S) TestRedisFlushDB(c *check.C) {
	key := test.RandStr()
	val := test.RandStr()

	c.Assert(client.Set(key, val), check.IsNil)

	c.Assert(client.FlushDB(), check.IsNil)

	result, err := client.GetString(key)
	c.Assert(err, check.IsNil)
	c.Assert(result, check.Equals, "")
}

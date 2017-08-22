#!/usr/bin/env python
#
# Copyright 2015-2016 Applatix, Inc.  All rights reserved.
#
# -*- coding: utf-8 -*-
#

from . import db


class Configuration(db.Model):
    id = db.Column(db.String(200), primary_key=True)
    nickname = db.Column(db.String(200), index=True, unique=True)
    type = db.Column(db.String(100))
    config = db.Column(db.String(5000))
    active = db.Column(db.Boolean)

    def __repr__(self):
        return '<SMTP Configuration %r>' % self.nickname

    def as_dict(self):
        return {c.name: getattr(self, c.name) for c in self.__table__.columns}

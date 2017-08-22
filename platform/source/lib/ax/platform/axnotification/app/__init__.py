from flask import Flask
from flask_sqlalchemy import SQLAlchemy
from ax.platform.axnotification import config

myapp = Flask(__name__)
myapp.config.from_object('ax.platform.axnotification.config')
db = SQLAlchemy(myapp)

from . import views, models

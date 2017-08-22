import os
import sys

SQLALCHEMY_DATABASE_URI = 'sqlite://'
SQLALCHEMY_TRACK_MODIFICATIONS = False

DEBUG = False

# Logging Settings
LOG_LEVEL = os.environ.get('FLASK_LOGLEVEL') or ('DEBUG' if DEBUG else 'INFO')
LOGGER_NAME = 'ax.notification'

LOGGING = {
    'version': 1,
    'disable_existing_loggers': False,
    'formatters': {
        'verbose': {
            'format': "[%(asctime)s::%(levelname)s::%(name)s::%(lineno)s]  %(message)s",
            'datefmt': "%Y-%m-%dT%H:%M:%S"
        }
    },
    'handlers': {
        'console': {
            'level': LOG_LEVEL,
            'class': 'logging.StreamHandler',
            'stream': sys.stdout,
            'formatter': 'verbose'
        },
    },
    'loggers': {
        LOGGER_NAME: {
            'handlers': ['console'],
            'level': LOG_LEVEL,
            'propagate': True,
        },
        'flask': {
            'handlers': ['console'],
            'level': LOG_LEVEL,
            'propagate': True,
        }
    }
}

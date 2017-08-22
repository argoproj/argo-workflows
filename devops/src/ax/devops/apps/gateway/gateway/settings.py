#!/usr/bin/env python
#
# Copyright 2015-2017 Applatix, Inc.  All rights reserved.
#
# -*- coding: utf-8 -*-
#
import os
import sys

# Standard default Django settings

BASE_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

SECRET_KEY = 'y&h50xpbsgujool%k2w7w-7c$i8dhqub3dr$l@h&3$^cx!uvzi'

MIDDLEWARE_CLASSES = [
    'django.middleware.security.SecurityMiddleware',
    'django.contrib.sessions.middleware.SessionMiddleware',
    'django.middleware.common.CommonMiddleware',
    'django.middleware.csrf.CsrfViewMiddleware',
    'django.contrib.auth.middleware.AuthenticationMiddleware',
    'django.contrib.auth.middleware.SessionAuthenticationMiddleware',
    'django.contrib.messages.middleware.MessageMiddleware',
    'django.middleware.clickjacking.XFrameOptionsMiddleware',
]

TEMPLATES = [
    {
        'BACKEND': 'django.template.backends.django.DjangoTemplates',
        'DIRS': [],
        'APP_DIRS': True,
        'OPTIONS': {
            'context_processors': [
                'django.template.context_processors.debug',
                'django.template.context_processors.request',
                'django.contrib.auth.context_processors.auth',
                'django.contrib.messages.context_processors.messages',
            ],
        },
    },
]

AUTH_PASSWORD_VALIDATORS = [
    {
        'NAME': 'django.contrib.auth.password_validation.UserAttributeSimilarityValidator',
    },
    {
        'NAME': 'django.contrib.auth.password_validation.MinimumLengthValidator',
    },
    {
        'NAME': 'django.contrib.auth.password_validation.CommonPasswordValidator',
    },
    {
        'NAME': 'django.contrib.auth.password_validation.NumericPasswordValidator',
    },
]

WSGI_APPLICATION = 'gateway.wsgi.application'

LANGUAGE_CODE = 'en-us'

USE_I18N = True

USE_L10N = True

USE_TZ = True

# AX customized settings

DEBUG = False

ALLOWED_HOSTS = ['*']

# AX home
AXHOME = os.environ['AXHOME']

# Database settings

DATABASES = {
    'default': {
        'ENGINE': 'django.db.backends.sqlite3',
        'NAME': os.path.join(AXHOME, 'db', 'db.sqlite3'),
    }
}

# Root URL configuration

ROOT_URLCONF = 'gateway.urls'

# Time zone

TIME_ZONE = 'UTC'

# Static file URL
STATIC_URL = '/static/'

# Applications

INSTALLED_APPS = [
    'django.contrib.admin',
    'django.contrib.auth',
    'django.contrib.contenttypes',
    'django.contrib.sessions',
    'django.contrib.messages',
    'django.contrib.staticfiles',
    'django_crontab',
    'django_extensions',
    'django_filters',
    'axjira',
    'rest_framework',
    'result',
    'scm',
]

# Date / time format

DATETIME_FORMAT = 'c'

DATE_FORMAT = 'Y-m-d'

TIME_FORMAT = 'H:i:s'

# Logging

LOG_ROOT = os.path.join(AXHOME, 'log')

LOG_LEVEL = os.environ.get('DJANGO_LOGLEVEL') or ('DEBUG' if DEBUG else 'INFO')

LOGGER_NAME = 'ax.devops.gateway'

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
        'django': {
            'handlers': ['console'],
            'level': LOG_LEVEL,
            'propagate': True,
        }
    }
}

# REST framework settings

REST_FRAMEWORK = {
    'DEFAULT_AUTHENTICATION_CLASSES': (
        'rest_framework.authentication.SessionAuthentication',
    ),
    'DEFAULT_PERMISSION_CLASSES': (
        'rest_framework.permissions.AllowAny',
    ),
    'DEFAULT_FILTER_BACKENDS': (
        'rest_framework.filters.DjangoFilterBackend',
        'rest_framework.filters.OrderingFilter',
    ),
    'DEFAULT_RENDERER_CLASSES': (
        'rest_framework.renderers.JSONRenderer',
    ),
    'DEFAULT_PAGINATION_CLASS': 'rest_framework.pagination.LimitOffsetPagination',
    'DEFAULT_VERSIONING_CLASS': 'rest_framework.versioning.NamespaceVersioning',
    'EXCEPTION_HANDLER': 'gateway.exceptions.ax_exception_handler'
}

CRONJOBS = [
    ('*/15 * * * *', 'axjira.cron.check_github_whitelist', '>> /var/log/cron.log 2>&1')
]
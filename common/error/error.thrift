// Copyright 2015-2016 Applatix, Inc. All rights reserved.
namespace go axerror
namespace py axerror
namespace js axerror

const i32 REST_STATUS_OK = 200;
const i32 REST_CREATE_OK = 201;
const i32 REST_BAD_REQ = 400;
const i32 REST_AUTH_DENIED = 401;
const i32 REST_FORBIDDEN = 403;
const i32 REST_NOT_FOUND = 404;
const i32 REST_INTERNAL_ERR = 500;

struct AXError {
	1: required string code;
	2: required string message;	
	3: required string detail;
}

// Core AX Error codes "ERR_AX_XXXX"
// We use it in the case of internal AXError (e.g. parsing failures, or protocol failures).
const AXError ERR_AX_INTERNAL = { 'code':"ERR_AX_INTERNAL", 'message':"An internal error has happened." }
const AXError ERR_AX_TIMEOUT = { 'code':"ERR_AX_TIMEOUT", 'message':"Requested operation exceeded maximum timeout." }
const AXError ERR_AX_ILLEGAL_ARGUMENT = { 'code':"ERR_AX_ILLEGAL_ARGUMENT", 'message':"An illegal or inappropriate argument was supplied." }
const AXError ERR_AX_ILLEGAL_OPERATION = { 'code':"ERR_AX_ILLEGAL_OPERATION", 'message':"Requested operation invoked at an illegal or inappropriate time." }
const AXError ERR_AX_HTTP_CONNECTION = { 'code':"ERR_AX_HTTP_CONNECTION", 'message':"Failed to establish the http connection." }

// Client API Error Codes "ERR_API_XXXX"
const AXError ERR_API_INVALID_SESSION = { 'code':"ERR_API_INVALID_SESSION", 'message':"The session id %s is invalid." }
const AXError ERR_API_AUTH_FAILED = { 'code':"ERR_API_AUTH_FAILED", 'message':"The username/password is invalid." }
const AXError ERR_API_INVALID_PARAM = { 'code':"ERR_API_INVALID_PARAM" }
const AXError ERR_API_INVALID_REQ = { 'code':"ERR_API_INVALID_REQ" }
const AXError ERR_API_DUP_USERNAME = { 'code':"ERR_API_DUP_USERNAME", 'message':"The user name has been taken." }
const AXError ERR_API_DUP_GROUPNAME = { 'code':"ERR_API_DUP_GROUPNAME", 'message':"The group name has been taken." }
const AXError ERR_API_DUP_LABEL = { 'code':"ERR_API_DUP_LABEL", 'message':"The label name has been taken." }
const AXError ERR_API_INVALID_USERNAME = { 'code':"ERR_API_INVALID_USERNAME", 'message':"The username is not a valid email address." }
const AXError ERR_API_EXPIRED_SESSION = { 'code':"ERR_API_EXPIRED_SESSION" }
const AXError ERR_API_FORBIDDEN_REQ = { 'code':"ERR_API_FORBIDDEN_REQ" }
const AXError ERR_API_RESOURCE_NOT_FOUND = { 'code':"ERR_API_RESOURCE_NOT_FOUND" }
const AXError ERR_API_WEAK_PASSWORD = {'code':"ERR_API_WEAK_PASSWORD", 'message':"The password strength is too weak."}
const AXError ERR_API_INTERNAL_ERROR = {'code':"ERR_API_INTERNAL_ERROR", 'message':"An internal error has happened."}

const AXError ERR_API_ACCOUNT_NOT_CONFIRMED = {'code':"ERR_API_ACCOUNT_NOT_CONFIRMED"}
const AXError ERR_API_AUTH_SAML_MISS_USERNAME = {'code':"ERR_API_AUTH_SAML_MISS_USERNAME"}
const AXError ERR_API_AUTH_SAML_MISS_CONFIG = {'code':"ERR_API_AUTH_SAML_MISS_CONFIG"}
const AXError ERR_API_AUTH_SAML_CREATE_REQ_FAILED = {'code':"ERR_API_AUTH_SAML_CREATE_REQ_FAILED"}
const AXError ERR_API_AUTH_SAML_INVALID_RESPONSE = {'code':"ERR_API_AUTH_SAML_INVALID_RESPONSE"}
const AXError ERR_API_AUTH_SAML_DECRYPTION_FAILED = {'code':"ERR_API_AUTH_SAML_DECRYPTION_FAILED"}
const AXError ERR_API_AUTH_PERMISSION_DENIED = {'code':"ERR_API_AUTH_PERMISSION_DENIED"}

// AX DB Error Codes "ERR_AXDB_XXXX"
const AXError ERR_AXDB_INTERNAL = { 'code':"ERR_AXDB_INTERNAL" }
const AXError ERR_AXDB_INVALID_PARAM = { 'code':"ERR_AXDB_INVALID_PARAM" }
const AXError ERR_AXDB_CONDITIONAL_UPDATE_FAILURE = { 'code':"ERR_AXDB_CONDITIONAL_UPDATE_FAILURE" }
const AXError ERR_AXDB_CONDITIONAL_UPDATE_FAILURE_NOT_EXIST = { 'code': "ERR_AXDB_CONDITIONAL_UPDATE_FAILURE_NOT_EXIST"}
const AXError ERR_AXDB_AUTH_FAILED = { 'code':"ERR_AXDB_AUTH_FAILED", 'message':"The request doesn't have a valid access key" }
const AXError ERR_AXDB_INSERT_DUPLICATE = { 'code':"ERR_AXDB_INSERT_DUPLICATE", 'message':"The request tries to insert an entry that already exists" }
const AXError ERR_AXDB_TABLE_NOT_FOUND = { 'code':"ERR_AXDB_TABLE_NOT_FOUND", 'message':"The request is for a table that doesn't exist" }

// AX Event error codes "ERR_EVENT_XXX"
const AXError ERR_EVENT_INVALID = { 'code':"ERR_EVENT_INVALID" }


// DevOps Error Codes "ERR_DEVOPS_XXXX"
const AXError ERR_AX_WORKFLOW_ALREADY_FAILED = { 'code':"ERR_AX_WORKFLOW_ALREADY_FAILED"}
const AXError ERR_AX_WORKFLOW_ALREADY_SUCCEED = { 'code':"ERR_AX_WORKFLOW_ALREADY_SUCCEED"}
const AXError ERR_AX_WORKFLOW_NOT_EXIST = { 'code':"ERR_AX_WORKFLOW_NOT_EXIST"}
const AXError ERR_AX_SERVICE_TEMPORARILY_UNAVAILABLE = { 'code':"ERR_AX_SERVICE_TEMPORARILY_UNAVAILABLE"}


// Platform Error Codes "ERR_PLAT_XXXX"
 







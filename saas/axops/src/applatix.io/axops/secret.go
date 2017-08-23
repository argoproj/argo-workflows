package axops

import (
	"applatix.io/axerror"
	"applatix.io/axops/secret"
	"applatix.io/axops/session"
	"applatix.io/axops/tool"
	"applatix.io/axops/utils"
	"crypto/rsa"
	"github.com/gin-gonic/gin"
	"time"
	//"github.com/kubernetes/client-go/kubernetes"
	//"github.com/kubernetes/client-go/rest"
	//"fmt"
	"crypto/x509"
	"encoding/base64"
)

// Fake Object to make swagger happy
type SecretObject struct {
	PlainData map[string]interface{} `json:"plain_text,omitempty"`
}

type EncryptedObject struct {
	CipherData map[string]interface{} `json:"cipher_text,omitempty"`
}

type MapObject map[string]interface{}

func SecretDecryptHandler(c *gin.Context) {

	//config, err := rest.InClusterConfig()
	//if err != nil {
	//	panic(err.Error())
	//}
	// creates the clientset
	//clientset, err := kubernetes.NewForConfig(config)
	//if err != nil {
	//	panic(err.Error())
	//}

	var payload map[string]interface{}
	err := utils.GetUnmarshalledBody(c, &payload)
	if err != nil {
		c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage("Request body doesn't contain a valid secret cipher text, err: "+err.Error()))
		return
	}

	cipherdata, exist := payload[secret.SECRET_PAYLOAD_CIPHERTEXT]
	if !exist {
		c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage("Request body doesn't contain a valid secret cipher text format, err: "+err.Error()))
		return
	}
	plaintext, axErr := secret.DecryptSecret(cipherdata.(map[string]interface{}))

	if axErr != nil {
		// no further error information would be provided
		//It is deliberately vague to avoid adaptive attacks.
		c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage("Encrypted secret can't be decrypted correctly"))

	} else {
		payload := map[string]interface{}{
			secret.SECRET_PAYLOAD_PLAINTEXT: map[string]interface{}{
				secret.SECRET_PAYLOAD_DECRYPT: plaintext,
			},
		}
		c.JSON(axerror.REST_STATUS_OK, payload)
	}
}

// @Title Encrypt a secret
// @Description Encrypt a secret
// @Accept  json
// @Param   secret       body    SecretObject	true        "secret object"
// @Success 200 {object} EncryptedObject
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /secret
// @Router /secret/encrypt [POST]
func SecretEncryptHandler(c *gin.Context) {
	var payload map[string]interface{}
	err := utils.GetUnmarshalledBody(c, &payload)
	if err != nil {
		c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage("Request body doesn't contain a valid secret plain text, err: "+err.Error()))
		return
	}
	plaindata, exist := payload[secret.SECRET_PAYLOAD_PLAINTEXT]
	if !exist {
		c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage("Request body doesn't contain a valid secret plain text format, err: "+err.Error()))
		return
	}
	ciphertext, axErr := secret.EncryptSecret(plaindata.(map[string]interface{}))

	if axErr != nil {
		c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage("Secret wasn't encrypted correctly, err: "+axErr.Error()))

	} else {
		payload := map[string]interface{}{
			secret.SECRET_PAYLOAD_CIPHERTEXT: map[string]interface{}{
				secret.SECRET_PAYLOAD_ENCRYPT: ciphertext,
			},
		}
		c.JSON(axerror.REST_STATUS_OK, payload)
	}
}

// @Title Download key used for encryption/decryption
// @Description Download key used for encryption/decryption
// @Accept  json
// @Param   session      body    MapObject	true        "session information"
// @Success 200 {object} MapObject
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /secret
// @Router /secret/key [POST]
func SecretDownloadKeyHandler(c *gin.Context) {
	var payload map[string]interface{}
	err := utils.GetUnmarshalledBody(c, &payload)
	if err != nil {
		c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage("Request body is invalid, err: "+err.Error()))
		return
	}
	sessionID, exist := payload["session"]
	if !exist {
		c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage("Request body doesn't contain session information."))
		return
	}
	ssn, axErr := session.GetSessionById(sessionID.(string))
	if axErr != nil {
		c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage("Failed to get session, err: "+axErr.Error()))
		return
	}

	// if 5 minutes has passed since last login, we will reject the download operation
	if ssn.Ctime+5*60 < time.Now().Unix() {
		c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage("5 minutes has passed since last login, need to login again."))
		return
	}

	//if secret.RSAKey == nil {
	//	secret.LoadRSAKey()
	//}

	keyBytes := x509.MarshalPKCS1PrivateKey(secret.RSAKey)
	c.JSON(axerror.REST_STATUS_OK, map[string]interface{}{
		secret.SECRET_PLAYLOAD_RSAKEY: base64.StdEncoding.EncodeToString(keyBytes),
	})
}

// @Title Update key used for encryption/decryption
// @Description Update key used for encryption/decryption
// @Accept  json
// @Param   key      body    MapObject	true        "Key information"
// @Success 200 {object} MapObject
// @Failure 400 {object} axerror.AXError "Invalid request"
// @Failure 500 {object} axerror.AXError "Internal server error"
// @Resource /secret
// @Router /secret/key [PUT]
func SecretUpdateKeyHandler(c *gin.Context) {
	var payload map[string]interface{}
	var newKey *rsa.PrivateKey
	var keyConfig *tool.SecureKeyConfig
	err := utils.GetUnmarshalledBody(c, &payload)
	if err != nil {
		c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage("Request body is invalid, err: "+err.Error()))
		return
	}

	keyVal, exist := payload["key"]
	if !exist {
		c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage("Request body doesn't contain key."))
		return
	}

	decoded, err := base64.StdEncoding.DecodeString(keyVal.(string))
	if err != nil {
		c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage("Secure Key cannot be decoded."))
	}
	//newKey = keyVal.(*rsa.PrivateKey)
	newKey, err = x509.ParsePKCS1PrivateKey(decoded)
	//err = json.Unmarshal([]byte(keyVal.(string)), &newKey)
	if err != nil {
		c.JSON(axerror.REST_BAD_REQ, axerror.ERR_API_INVALID_PARAM.NewWithMessage("Secure key is invalid, err: "+err.Error()))
		return
	}

	// get old key from db
	keys, axErr := tool.GetToolsByType(tool.TypeSecureKey)
	if axErr != nil {
		c.JSON(axerror.REST_INTERNAL_ERR, axerror.ERR_API_INTERNAL_ERROR.NewWithMessagef("Failed to load the Secure Key from DB:", axErr))
		return
	}

	if len(keys) > 1 {
		c.JSON(axerror.REST_INTERNAL_ERR, axerror.ERR_AX_INTERNAL.NewWithMessagef("There should be only one key allowed, we got %d", len(keys)))
		return
	}

	if len(keys) == 0 {
		base := &tool.ToolBase{
			Category: tool.CategorySecret,
			Type:     tool.TypeSecureKey,
		}
		keyConfig = &tool.SecureKeyConfig{base, newKey, "default", secret.SECRET_KEY_VERSION}
	} else {
		keyConfig = keys[0].(*tool.SecureKeyConfig)
		keyConfig.PrivateKey = newKey
		keyConfig.Version = secret.SECRET_KEY_VERSION
	}
	axErr, _ = tool.Update(keyConfig)
	if axErr != nil {
		c.JSON(axerror.REST_INTERNAL_ERR, axerror.ERR_API_INTERNAL_ERROR.NewWithMessagef("Failed to persist the updated RSA secure key: %v", axErr))
		return
	}

	//update the key in cache
	secret.UpdateRSAKeyInCache(newKey)
	c.JSON(axerror.REST_STATUS_OK, nullMap)
}

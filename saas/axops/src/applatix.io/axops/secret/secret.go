package secret

import (
	"applatix.io/axerror"
	"applatix.io/axops/tool"
	"applatix.io/axops/utils"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	//"github.com/gin-gonic/gin"
	//"github.com/kubernetes/client-go/kubernetes"
	//"github.com/kubernetes/client-go/rest"
	"strings"
	"sync"
)

var RSAKey *rsa.PrivateKey
var RSAKeyMutex sync.Mutex

//var K8sCl *kubernetes.Clientset

func Init() *axerror.AXError {
	axErr := CreateRSAKey()
	if axErr != nil {
		return axErr
	}
	return CreateK8sClient()

}

func CreateK8sClient() *axerror.AXError {
	// creates the in-cluster config
	//config, err := rest.InClusterConfig()
	//if err != nil {
	//	panic(err.Error())
	//}
	//// creates the clientset
	//client, err := kubernetes.NewForConfig(config)
	//if err != nil {
	//	panic(err.Error())
	//}
	//K8sCl = client
	return nil
}

func CreateRSAKey() *axerror.AXError {
	size := 2569
	key, err := rsa.GenerateKey(rand.Reader, size)
	if err != nil {
		return axerror.ERR_AX_INTERNAL.NewWithMessagef("failed to generate RSA key: %s", err)
	}

	// we cache it in memory to avoid checking with db.
	UpdateRSAKeyInCache(key)
	return nil
}

func UpdateRSAKeyInCache(key *rsa.PrivateKey) *axerror.AXError {
	RSAKeyMutex.Lock()
	defer RSAKeyMutex.Unlock()
	RSAKey = key
	return nil
}

// load RSAKey from axdb and cache it to avoid checking with db for each encryption/decryption
// If it doesn't exist yet, just create it.
func LoadRSAKey() *axerror.AXError {
	var axErr *axerror.AXError = nil
	keys, axErr := tool.GetToolsByType(tool.TypeSecureKey)
	if axErr != nil {
		return axerror.ERR_API_INTERNAL_ERROR.NewWithMessagef("Failed to load the Secure Key from DB:", axErr)
	}

	if len(keys) == 0 {
		utils.InfoLog.Println("There is no key in use. The server will start generating the RSA secure key.")
		CreateRSAKey()
		base := &tool.ToolBase{
			Category: tool.CategorySecret,
			Type:     tool.TypeSecureKey,
			URL:      "default",
		}

		keyConfig := &tool.SecureKeyConfig{base, RSAKey, "default", SECRET_KEY_VERSION}
		axErr, _ := tool.Create(keyConfig)
		if axErr != nil {
			panic(fmt.Sprintf("Failed to persist the newly created RSA secure key: %v", axErr))
		}
	} else if len(keys) != 1 {
		axErr = axerror.ERR_API_INTERNAL_ERROR.NewWithMessagef("There should be only one key allowed, we got %d", len(keys))
	} else {
		keyconfig := keys[0].(*tool.SecureKeyConfig)
		//we need to update the key
		if keyconfig.Version != SECRET_KEY_VERSION {
			CreateRSAKey()
			keyconfig.PrivateKey = RSAKey
			keyconfig.Version = SECRET_KEY_VERSION
			axErr, _ = tool.Update(keyconfig)
			if axErr != nil {
				return axerror.ERR_API_INTERNAL_ERROR.NewWithMessagef("Failed to persist the updated RSA secure key: %v", axErr)
			}
		}
		UpdateRSAKeyInCache(keyconfig.PrivateKey)
	}
	return axErr
}

func DecryptSecret(payload map[string]interface{}) (string, *axerror.AXError) {
	var ciphertext interface{}
	var repo interface{}
	var exist bool
	ciphertext, exist = payload[SECRET_CIPHERTEXT]
	if !exist {
		return "", axerror.ERR_API_INVALID_PARAM.NewWithMessage("Request body doesn't contain valid secret cipher text")
	}

	segs := strings.Split(strings.TrimSuffix(strings.TrimPrefix(ciphertext.(string), "=="), "=="), "$")
	if len(segs) != 5 {
		return "", axerror.ERR_API_INVALID_PARAM.NewWithMessage("Format of cipher text is invalid.")
	}

	repo, exist = payload[SECRET_REPONAME]
	if !exist {
		return "", axerror.ERR_API_INVALID_PARAM.NewWithMessage("Request body doesn't contain valid secret cipher text")
	}

	decoded, dErr := base64.StdEncoding.DecodeString(segs[4])
	if dErr != nil {
		return "", axerror.ERR_API_INVALID_PARAM.NewWithMessagef("Cipher cannot be decoded: %s\n", dErr)
	}
	plaintext, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, RSAKey, []byte(decoded), []byte(repo.(string)))

	if err != nil {
		return "", axerror.ERR_API_INVALID_PARAM.NewWithMessagef("Error from decryption: %s\n", err)
	} else {
		return string(plaintext), nil
	}
}

func EncryptSecret(payload map[string]interface{}) (string, *axerror.AXError) {
	var plaintext interface{}
	var repo interface{}
	var exist bool

	plaintext, exist = payload[SECRET_PLAINTEXT]
	if !exist {
		return "", axerror.ERR_API_INVALID_PARAM.NewWithMessage("Request body doesn't contain valid secret")
	}

	repo, exist = payload[SECRET_REPONAME]
	if !exist {
		return "", axerror.ERR_API_INVALID_PARAM.NewWithMessage("Request body doesn't contain valid secret")
	}

	k := (RSAKey.PublicKey.N.BitLen() + 7) / 8
	hashlen := sha256.New().Size()
	limit := k - 2*hashlen + 2
	utils.InfoLog.Printf("[Secret]: k=%d, hashlen=%d, limit = %d, text len = %d", k, hashlen, limit, len(plaintext.(string)))
	ciphertext, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, &RSAKey.PublicKey, []byte(plaintext.(string)), []byte(repo.(string)))
	if err != nil {
		return "", axerror.ERR_API_INVALID_PARAM.NewWithMessagef("Error from encryption: %s\n", err)
	} else {
		encoded := "%%secrets.==$1$key=default$text$" + base64.StdEncoding.EncodeToString(ciphertext) + "==%%"
		return encoded, nil
	}
}

//func VerifyNameSpace(c *gin.Context) bool {
//	callerIp := c.Request.RemoteAddr
//	utils.InfoLog.Printf("[Secret] Caller IP: %s", callerIp)
//
//	pods, err := K8sCl.CoreV1().Pods("").List(metav1.ListOptions{})
//	if err != nil {
//		panic(err.Error())
//	}
//	fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))
//
//}

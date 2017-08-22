package secret_test

import (
	"applatix.io/axerror"
	"applatix.io/axops/secret"
	"gopkg.in/check.v1"
)

func (s *S) TestCryption(c *check.C) {
	//generate private key
	err := secret.CreateRSAKey()
	c.Assert(err, check.IsNil)

	var plaintext string = "applatix_secret_management_test"
	var ciphertext string = ""
	var repo string = "real_repo"
	var fakerepo string = "fake_repo"
	var axErr *axerror.AXError
	var decryptedtext string

	// an invalid payload to encrypt: the key for plaintext isn't SECRET_CIPHERTEXT
	payload := map[string]interface{}{
		secret.SECRET_CIPHERTEXT: plaintext,
		secret.SECRET_REPONAME:   repo,
	}

	ciphertext, axErr = secret.EncryptSecret(payload)
	c.Assert(axErr, check.Not(check.Equals), nil)

	// an invalid payload to encrypt: the plain text is missing
	payload = map[string]interface{}{
		secret.SECRET_PLAINTEXT: plaintext,
	}
	ciphertext, axErr = secret.EncryptSecret(payload)
	c.Assert(axErr, check.Not(check.Equals), nil)

	// an invalid payload to encrypt: the repo is missing
	payload = map[string]interface{}{
		secret.SECRET_PLAINTEXT: plaintext,
	}

	ciphertext, axErr = secret.EncryptSecret(payload)
	c.Assert(axErr, check.Not(check.Equals), nil)

	// a valid payload to encrypt
	payload = map[string]interface{}{
		secret.SECRET_PLAINTEXT: plaintext,
		secret.SECRET_REPONAME:  repo,
	}
	ciphertext, axErr = secret.EncryptSecret(payload)
	c.Assert(axErr, check.IsNil)

	// use the same repo name to decrypt the cipher text, the decryptedtext should be the same as plaintext
	payload = map[string]interface{}{
		secret.SECRET_CIPHERTEXT: ciphertext,
		secret.SECRET_REPONAME:   repo,
	}
	decryptedtext, axErr = secret.DecryptSecret(payload)
	c.Assert(axErr, check.IsNil)
	c.Assert(plaintext, check.Equals, decryptedtext)

	// use different repo name to decrypt the cipher text, we should get an error
	payload = map[string]interface{}{
		secret.SECRET_CIPHERTEXT: ciphertext,
		secret.SECRET_REPONAME:   fakerepo,
	}
	decryptedtext, axErr = secret.DecryptSecret(payload)
	c.Assert(axErr, check.Not(check.Equals), nil)

	// an invalid payload to decrypt: no cipher text is specified
	payload = map[string]interface{}{
		secret.SECRET_REPONAME: repo,
	}
	decryptedtext, axErr = secret.DecryptSecret(payload)
	c.Assert(axErr, check.Not(check.Equals), nil)

	// an invalid payload to decrypt: no repo is specified
	payload = map[string]interface{}{
		secret.SECRET_CIPHERTEXT: ciphertext,
	}
	decryptedtext, axErr = secret.DecryptSecret(payload)
	c.Assert(axErr, check.Not(check.Equals), nil)
}

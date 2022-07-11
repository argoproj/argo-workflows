package azure

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetermineAccountName(t *testing.T) {
	validUrls := []string{
		"https://accountname/",
		"https://accountname.blob.core.windows.net",
		"https://accountname.blob.core.windows.net/",
		"https://accountname.blob.core.windows.net:1234/",
		"https://localhost/accountname/foo",
		"https://127.0.0.1/accountname/foo",
		"https://localhost:1234/accountname/foo",
		"https://127.0.0.1:1234/accountname/foo",
	}
	for _, u := range validUrls {
		u, err := url.Parse(u)
		assert.NoError(t, err)
		accountName, err := determineAccountName(u)
		assert.NoError(t, err)
		assert.Equal(t, "accountname", accountName)
	}

	invalidUrls := []string{
		"https://127.0.0.1/foo",
	}
	for _, u := range invalidUrls {
		u, err := url.Parse(u)
		assert.NoError(t, err)
		accountName, err := determineAccountName(u)
		assert.Error(t, err)
		assert.Equal(t, "", accountName)
	}
}
